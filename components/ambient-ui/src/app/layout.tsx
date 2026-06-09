import type { Metadata } from "next"
import { Red_Hat_Text, Red_Hat_Mono } from "next/font/google"
import { ThemeProvider } from "@/components/providers/theme-provider"
import { QueryProvider } from "@/components/providers/query-provider"
import { Toaster } from "@/components/ui/sonner"
import "./globals.css"

const redHatText = Red_Hat_Text({
  variable: "--font-rh-text",
  subsets: ["latin"],
})

const redHatMono = Red_Hat_Mono({
  variable: "--font-rh-mono",
  subsets: ["latin"],
})

export const metadata: Metadata = {
  title: "ACP",
  description: "Agent Control Plane operations console",
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body
        className={`${redHatText.variable} ${redHatMono.variable} font-sans min-h-screen antialiased`}
      >
        <ThemeProvider
          attribute="class"
          defaultTheme="system"
          enableSystem
          disableTransitionOnChange
        >
          <QueryProvider>
            {children}
            <Toaster />
          </QueryProvider>
        </ThemeProvider>
      </body>
    </html>
  )
}
