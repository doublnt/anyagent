import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "AnyAgent - Agent Enhancement Platform",
  description: "Enhance your Claude Code / Codex / Cursor with Agent Packs, Memory, and Traces",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col bg-gray-50">
        <nav className="bg-white border-b border-gray-200">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex justify-between h-16">
              <div className="flex items-center">
                <a href="/" className="text-xl font-bold text-gray-900">
                  AnyAgent
                </a>
                <div className="ml-10 flex space-x-8">
                  <a
                    href="/store"
                    className="text-gray-500 hover:text-gray-900 inline-flex items-center px-1 pt-1 text-sm font-medium"
                  >
                    Store
                  </a>
                  <a
                    href="/dashboard"
                    className="text-gray-500 hover:text-gray-900 inline-flex items-center px-1 pt-1 text-sm font-medium"
                  >
                    Dashboard
                  </a>
                  <a
                    href="/traces"
                    className="text-gray-500 hover:text-gray-900 inline-flex items-center px-1 pt-1 text-sm font-medium"
                  >
                    Traces
                  </a>
                  <a
                    href="/memory"
                    className="text-gray-500 hover:text-gray-900 inline-flex items-center px-1 pt-1 text-sm font-medium"
                  >
                    Memory
                  </a>
                </div>
              </div>
              <div className="flex items-center">
                <a
                  href="/login"
                  className="text-gray-500 hover:text-gray-900 text-sm font-medium"
                >
                  Sign in
                </a>
              </div>
            </div>
          </div>
        </nav>
        <main className="flex-1">{children}</main>
      </body>
    </html>
  );
}
