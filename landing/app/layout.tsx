import type { Metadata, Viewport } from "next";
import localFont from "next/font/local";
import "./globals.css";

const geist = localFont({
  src: "./fonts/geist-var.woff2",
  variable: "--font-geist",
  weight: "100 900",
  display: "swap",
});

const geistMono = localFont({
  src: "./fonts/geist-mono-var.woff2",
  variable: "--font-geist-mono",
  weight: "100 900",
  display: "swap",
});

export const metadata: Metadata = {
  metadataBase: new URL("https://dash-homelab.vercel.app"),
  title: "Dash - Self-hosted homelab dashboard",
  description:
    "Service boards, uptime monitors, domain watch, and system metrics in one container. A single static Go binary with an embedded React UI. No YAML, no external database.",
  openGraph: {
    title: "Dash - Self-hosted homelab dashboard",
    description:
      "Service boards, uptime monitors, domain watch, and system metrics in one container. One static Go binary, no YAML.",
    type: "website",
    images: ["/dash.svg"],
  },
};

export const viewport: Viewport = {
  themeColor: "#0a0a0a",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" className={`${geist.variable} ${geistMono.variable}`}>
      <body className="font-sans">{children}</body>
    </html>
  );
}
