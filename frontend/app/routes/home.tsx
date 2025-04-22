import { useState } from "react";

import type { Route } from "./+types/home";
import { Spinner } from "~/components/Spinner";

export function meta({}: Route.MetaArgs) {
	return [
		{ title: "sshtalk" },
		{
			name: "description",
			content: "talk to your ssh, with AI powered chat, try ssh sshtalk.com",
		},
	];
}

function WelcomeMessage() {
	return (
		<div className="flex flex-col text-sm text-center h-full justify-center">
			<span>Welcome to sshtalk!</span>
			<span>Type a message and press Enter to send.</span>
		</div>
	);
}

interface Message {
	content: string;
	fromUser: boolean;
}

export default function Home() {
	const [message, setMessage] = useState("");
	const [messages, setMessages] = useState<Message[]>([]);
	const [tempMessage, setTempMessage] = useState("");
	const [isPending, setIsPending] = useState(false);

	return (
		<div className="p-2 flex flex-col gap-8 h-dvh">
			<div className="border-1 rounded-md p-2 flex-1 flex flex-col">
				{messages.length === 0 && !isPending && <WelcomeMessage />}
				{messages.map((message, index) => (
					<div
						key={index}
						className={`flex ${message.fromUser ? "justify-end" : "justify-start"}`}
					>
						<div
							className={`rounded-sm text-sm px-2.5 py-1.5 ${message.fromUser ? "border-1" : ""}`}
						>
							{message.content}
						</div>
					</div>
				))}
				{isPending && tempMessage === "" && (
					<div className="flex justify-start">
						<div className="rounded-sm text-sm px-2.5 py-1.5">
							Thinking <Spinner />
						</div>
					</div>
				)}
				{isPending && tempMessage !== "" && (
					<div className="flex justify-start">
						<div className="rounded-sm text-sm px-2.5 py-1.5">
							{tempMessage} <Spinner />
						</div>
					</div>
				)}
			</div>
			<div className="border-1 rounded-md flex items-center">
				<span className="text-sm px-1">{">"}</span>
				<textarea
					rows={1}
					value={message}
					onChange={(e) => setMessage(e.target.value)}
					onKeyDown={async (e) => {
						if (e.key === "Enter") {
							e.preventDefault();
							if (isPending) {
								return;
							}
							setMessage("");
							const content = message.trim();
							if (content === "") return;
							if (content === "/clear") {
								setMessages([]);
								return;
							}
							setIsPending(true);
							setMessages((prev) => [...prev, { content, fromUser: true }]);
							const openaiMessages = messages
								.map((message) => ({
									role: message.fromUser ? "user" : "assistant",
									content: message.content,
								}))
								.concat({ role: "user", content });
							const response = await fetch("/api/chat", {
								method: "POST",
								body: JSON.stringify(openaiMessages),
							});
							const stream = response.body?.getReader();
							if (!stream) return;
							let finalMessage = "";
							while (true) {
								const { done, value } = await stream.read();
								if (done) break;
								const text = new TextDecoder().decode(value);
								finalMessage += text;
								setTempMessage((prev) => prev + text);
							}
							setIsPending(false);
							setTempMessage("");
							setMessages((prev) => [
								...prev,
								{ content: finalMessage, fromUser: false },
							]);
						}
					}}
					className="flex-1 h-8 w-full text-sm resize-none outline-none flex items-center pt-1.5"
					placeholder="Send a message..."
				/>
			</div>
		</div>
	);
}
