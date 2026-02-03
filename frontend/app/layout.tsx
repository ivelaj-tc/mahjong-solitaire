import type { Metadata } from "next";
import { Cinzel, Manrope } from "next/font/google";
import Script from "next/script";
import "./globals.css";

const cinzel = Cinzel({
	variable: "--font-display",
	subsets: ["latin"],
	weight: ["400", "600", "700"],
});

const manrope = Manrope({
	variable: "--font-body",
	subsets: ["latin"],
	weight: ["400", "500", "600", "700"],
});

export const metadata: Metadata = {
	title: "Mahjong Push Arena",
	description: "A two-player mahjong-inspired tile pushing game.",
	creator: "GBP",
	keywords: ["mahjong", "push", "arena", "game", "two player", "mahjong inspired", "tile pushing",
		 "mahjong solitaire", "mahjong push arena", "mahjong push arena game", "mahjong push arena two player", "mahjong push arena mahjong inspired", "mahjong push arena tile pushing", "mahjong push arena mahjong solitaire"]
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  	return (
		<html lang="en">
			<body
				className={`${manrope.variable} ${cinzel.variable} antialiased`}
			>
				<Script src="/runtime-env.js" strategy="beforeInteractive" />
				{children}
			</body>
		</html>
	);
}
