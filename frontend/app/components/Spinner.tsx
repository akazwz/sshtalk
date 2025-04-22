import { useEffect, useState } from "react";

interface SpinnerProps {
	className?: string;
}

export function Spinner({ className = "" }: SpinnerProps) {
	const frames = ["⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "];
	const [frameIndex, setFrameIndex] = useState(0);

	useEffect(() => {
		const interval = setInterval(() => {
			setFrameIndex((prev) => (prev + 1) % frames.length);
		}, 100); // 100ms matches the Go version's FPS of time.Second/10

		return () => clearInterval(interval);
	}, []);

	return <span className={className}>{frames[frameIndex]}</span>;
}
