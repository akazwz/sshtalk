package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/ssestream"
)

const (
	thinkingText = "Thinking" // 提取为常量
	gap          = "\n\n"
	systemPrompt = `Do not use markdown except when user asks for it.`
	welcomeMsg   = `Welcome to sshtalk!
	Type a message and press Enter to send.`
)

var (
	openaiBaseURL = os.Getenv("OPENAI_BASE_URL")
	openaiAPIKey  = os.Getenv("OPENAI_API_KEY")
	openaiModel   = os.Getenv("OPENAI_MODEL")
)

func main() {
	// Add command line flags
	sshMode := flag.Bool("ssh", false, "Run as SSH server")
	flag.Parse()

	if *sshMode {
		// Start SSH server mode
		log.Println("Starting in SSH server mode...")
		startSSHServer()
	} else {
		// Start direct TUI mode
		m := initialModel()
		p := tea.NewProgram(&m, tea.WithAltScreen())

		// 在程序开始前设置程序引用
		m.program = p

		if _, err := p.Run(); err != nil {
			log.Fatal(err)
		}
	}
}

type (
	errMsg error
	// 添加自定义消息类型用于接收AI响应
	aiResponseMsg struct {
		content      string
		done         bool
		err          error
		nextChunkCmd tea.Cmd // 获取下一个块的命令
	}
)

type message struct {
	content  string
	fromUser bool
}

type model struct {
	openaiClient  openai.Client
	viewport      viewport.Model
	messages      []string                                 // 渲染后的消息（带样式）
	rawMessages   []message                                // 原始消息内容（不带样式）
	chatHistory   []openai.ChatCompletionMessageParamUnion // 聊天历史记录
	textarea      textarea.Model
	senderStyle   lipgloss.Style
	receiverStyle lipgloss.Style
	spinner       spinner.Model
	isWaiting     bool // 是否正在等待响应
	err           error
	program       *tea.Program // 添加程序引用
	lastMsgDone   bool         // 最后一条消息是否完成
	// 新增的样式对象
	userMsgStyle   lipgloss.Style
	userAlignStyle lipgloss.Style
	botMsgStyle    lipgloss.Style
	welcomeStyle   lipgloss.Style

	// 渲染缓存相关
	lastViewportWidth int    // 上次渲染时的视口宽度
	lastSpinnerFrame  string // 上次渲染时的spinner帧
	needsReformat     bool   // 是否需要重新格式化
}

func initialModel() model {

	openaiClient := openai.NewClient(option.WithBaseURL(openaiBaseURL), option.WithAPIKey(openaiAPIKey))

	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "> "

	ta.SetHeight(1)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	// Add border to textarea
	ta.FocusedStyle.Base = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	ta.BlurredStyle.Base = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())

	ta.ShowLineNumbers = false

	// Create a viewport that will be properly sized in the first Update
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder())
	// 注意：此时viewport尺寸还是0x0，实际的垂直居中会在第一次Update时处理
	vp.SetContent(lipgloss.NewStyle().Align(lipgloss.Center).Render(welcomeMsg))

	ta.KeyMap.InsertNewline.SetEnabled(false)

	// 初始化 spinner，使用Dot样式
	s := spinner.New()
	s.Spinner = spinner.Dot

	// 预先创建样式对象
	userMsgStyle := lipgloss.NewStyle().
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder()).
		Align(lipgloss.Right)

	rightAlignStyle := lipgloss.NewStyle().
		Align(lipgloss.Right).
		PaddingRight(1)

	botMsgStyle := lipgloss.NewStyle().
		Align(lipgloss.Left).
		PaddingLeft(1)

	welcomeStyle := lipgloss.NewStyle().
		Align(lipgloss.Center)

	return model{
		openaiClient: openaiClient,
		textarea:     ta,
		messages:     []string{},
		rawMessages:  []message{},
		chatHistory: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
		},
		viewport:       vp,
		senderStyle:    lipgloss.NewStyle(),
		receiverStyle:  lipgloss.NewStyle(),
		spinner:        s,
		isWaiting:      false,
		err:            nil,
		program:        nil,
		lastMsgDone:    true, // 初始状态为完成
		userMsgStyle:   userMsgStyle,
		userAlignStyle: rightAlignStyle,
		botMsgStyle:    botMsgStyle,
		welcomeStyle:   welcomeStyle,

		// 初始化渲染缓存相关字段
		lastViewportWidth: 0,
		lastSpinnerFrame:  "",
		needsReformat:     true,
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		cmds  []tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	cmds = append(cmds, tiCmd, vpCmd)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// 使 viewport 占据窗口大部分空间
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		m.viewport.Height = msg.Height - m.textarea.Height() - lipgloss.Height(gap)

		// 窗口大小变化，标记需要重新格式化
		if m.lastViewportWidth != m.viewport.Width {
			m.lastViewportWidth = m.viewport.Width
			m.needsReformat = true
		}

		if len(m.rawMessages) > 0 {
			// 窗口大小变化，重新格式化所有消息
			if m.needsReformat {
				m.formatMessages()
				m.needsReformat = false
			}
			m.viewport.GotoBottom()
		} else {
			// 只有欢迎消息居中显示
			welcomeMsg := welcomeMsg
			// 计算垂直居中所需的空行数
			msgLines := strings.Count(welcomeMsg, "\n") + 1
			padLines := (m.viewport.Height - msgLines) / 2
			if padLines > 0 {
				vertPadding := strings.Repeat("\n", padLines)
				welcomeMsg = vertPadding + welcomeMsg
			}

			contentStyle := m.welcomeStyle
			contentStyle = contentStyle.Width(m.viewport.Width)
			m.viewport.SetContent(contentStyle.Render(welcomeMsg))
		}
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case tea.KeyEnter:
			userMsg := m.textarea.Value()
			if userMsg != "" && !m.isWaiting {
				// 检查是否是清空历史的命令
				if userMsg == "/clear" {
					// 清空所有消息记录，只保留系统提示
					m.rawMessages = []message{}
					m.messages = []string{}
					m.chatHistory = []openai.ChatCompletionMessageParamUnion{
						openai.SystemMessage(systemPrompt),
					}

					// 重设视图
					m.viewport.SetContent(m.welcomeStyle.Render(welcomeMsg))
					m.textarea.Reset()
					return m, nil
				}

				// 添加用户原始消息到列表
				userContent := userMsg
				m.rawMessages = append(m.rawMessages, message{content: userContent, fromUser: true})

				// 添加到聊天历史
				m.chatHistory = append(m.chatHistory, openai.UserMessage(userMsg))

				// 标记需要重新格式化
				m.needsReformat = true

				// 先展示用户消息
				m.formatMessages()
				m.viewport.GotoBottom()

				// 添加一个加载中的消息
				m.isWaiting = true
				m.rawMessages = append(m.rawMessages, message{content: fmt.Sprintf("%s", thinkingText), fromUser: false})
				m.needsReformat = true
				m.formatMessages()
				m.viewport.GotoBottom()

				// 重置输入框
				m.textarea.Reset()

				// 创建一个命令函数来启动AI响应请求
				startAIRequest := func() tea.Cmd {
					return func() tea.Msg {
						// 创建context，可以在需要时取消
						ctx := context.Background()

						// 启动流式请求
						stream := m.openaiClient.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
							Model:    openaiModel,
							Messages: m.chatHistory,
						})

						// 创建新的响应处理器
						return fetchAIResponseCmd(stream)()
					}
				}

				return m, tea.Batch(
					m.spinner.Tick,
					startAIRequest(),
				)
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		m.isWaiting = false
		m.lastMsgDone = true
		return m, nil

	// 处理AI响应消息
	case aiResponseMsg:
		if msg.err != nil {
			m.err = msg.err
			m.isWaiting = false
			m.lastMsgDone = true
			return m, nil
		}

		// 标记需要重新格式化
		m.needsReformat = true

		// 移除加载中的消息
		if m.isWaiting && len(m.rawMessages) > 0 {
			// 移除最后一条"思考中"的消息
			m.rawMessages = m.rawMessages[:len(m.rawMessages)-1]
		}

		// 更新消息完成状态
		m.lastMsgDone = msg.done

		if msg.done {
			// 添加完整的AI响应到历史记录
			m.chatHistory = append(m.chatHistory, openai.AssistantMessage(msg.content))
			m.isWaiting = false

			// 删除现有的流式消息（如果有）
			if len(m.rawMessages) > 0 && !m.rawMessages[len(m.rawMessages)-1].fromUser {
				m.rawMessages = m.rawMessages[:len(m.rawMessages)-1]
			}

			// 添加消息，无需标记
			m.rawMessages = append(m.rawMessages, message{
				content:  msg.content,
				fromUser: false,
			})
		} else {
			// 流式更新
			if len(m.rawMessages) > 0 && !m.rawMessages[len(m.rawMessages)-1].fromUser {
				// 更新现有的bot消息内容
				m.rawMessages[len(m.rawMessages)-1].content = msg.content
			} else {
				// 添加新的bot消息
				m.rawMessages = append(m.rawMessages, message{
					content:  msg.content,
					fromUser: false,
				})
			}
		}

		// 重新格式化并滚动到底部
		m.formatMessages()
		m.needsReformat = false
		m.viewport.GotoBottom()

		// 如果有下一个命令，继续执行
		if !msg.done && msg.nextChunkCmd != nil {
			return m, msg.nextChunkCmd
		}

		return m, nil

	// 处理spinner tick
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		// 检查spinner外观是否变化
		currentFrame := m.spinner.View()
		spinnerChanged := m.lastSpinnerFrame != currentFrame
		m.lastSpinnerFrame = currentFrame

		// 如果正在等待且spinner变化，重新渲染
		if m.isWaiting && len(m.rawMessages) > 0 && spinnerChanged {
			m.needsReformat = true
			m.formatMessages()
			m.needsReformat = false
		}

		return m, cmd
	}

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	return fmt.Sprintf(
		"%s%s%s",
		m.viewport.View(),
		gap,
		m.textarea.View(),
	)
}

func (m *model) formatMessages() {
	// 如果不需要重新格式化，跳过
	if !m.needsReformat && len(m.messages) > 0 {
		return
	}

	m.messages = []string{}

	if len(m.rawMessages) == 0 {
		return
	}

	maxWidth := m.viewport.Width
	msgWidth := maxWidth * 3 / 4
	marginWidth := 1 // 减小边距到1个字符

	// 根据当前窗口大小更新样式的宽度属性
	userMsgStyle := m.userMsgStyle
	userMsgStyle = userMsgStyle.Width(msgWidth)

	userAlignStyle := m.userAlignStyle
	userAlignStyle = userAlignStyle.Width(maxWidth - marginWidth)

	botMsgStyle := m.botMsgStyle
	botMsgStyle = botMsgStyle.Width(msgWidth + marginWidth)

	for i, msg := range m.rawMessages {
		isLastMsg := i == len(m.rawMessages)-1
		displayContent := msg.content

		// 检查是否是最后一条正在加载的消息
		if isLastMsg && m.isWaiting && !msg.fromUser && strings.Contains(displayContent, thinkingText) {
			displayContent = fmt.Sprintf("%s %s", thinkingText, m.spinner.View())
		}

		// 如果是最后一条AI消息，并且消息尚未完成，添加spinner
		if isLastMsg && !msg.fromUser && !m.lastMsgDone {
			displayContent = fmt.Sprintf("%s %s", displayContent, m.spinner.View())
		}

		if msg.fromUser {
			// 用户消息样式（右侧）
			// 先渲染内容，不设置固定宽度
			contentStyle := lipgloss.NewStyle().
				Padding(0, 1).
				BorderStyle(lipgloss.RoundedBorder()).
				Align(lipgloss.Right).
				MaxWidth(msgWidth)

			// 渲染用户消息并右对齐
			formattedMsg := contentStyle.Render(displayContent)
			m.messages = append(m.messages, userAlignStyle.Render(formattedMsg))
		} else {
			// 机器人消息样式（左侧）- 使用预创建的样式
			m.messages = append(m.messages, botMsgStyle.Render(displayContent))
		}

		// 添加空行分隔消息
		m.messages = append(m.messages, "")
	}

	// 更新 viewport 内容
	m.viewport.SetContent(strings.Join(m.messages, "\n"))
}

// fetchAIResponseCmd 创建一个命令来获取下一个响应块
func fetchAIResponseCmd(stream *ssestream.Stream[openai.ChatCompletionChunk]) tea.Cmd {
	return fetchAIResponseCmdWithAccumulator(stream, &openai.ChatCompletionAccumulator{})
}

// fetchAIResponseCmdWithAccumulator 是fetchAIResponseCmd的辅助函数，接受一个累加器参数
func fetchAIResponseCmdWithAccumulator(stream *ssestream.Stream[openai.ChatCompletionChunk], acc *openai.ChatCompletionAccumulator) tea.Cmd {
	return func() tea.Msg {
		if stream.Next() {
			chunk := stream.Current()
			acc.AddChunk(chunk)

			if len(acc.Choices) > 0 {
				content := acc.Choices[0].Message.Content

				// 返回当前累积的内容和一个命令来获取下一部分
				return aiResponseMsg{
					content:      content,
					done:         false,
					err:          nil,
					nextChunkCmd: fetchAIResponseCmdWithAccumulator(stream, acc), // 传递同一个累加器
				}
			}
		}

		// 检查错误
		if err := stream.Err(); err != nil {
			return aiResponseMsg{
				content: "",
				done:    false,
				err:     err,
			}
		}

		// 流结束，返回完整内容
		content := ""
		if len(acc.Choices) > 0 {
			content = acc.Choices[0].Message.Content
		}

		return aiResponseMsg{
			content: content,
			done:    true,
			err:     nil,
		}
	}
}
