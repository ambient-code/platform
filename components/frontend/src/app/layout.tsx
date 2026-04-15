import type { Metadata } from "next";
import { GeistSans } from "geist/font/sans";
import { GeistMono } from "geist/font/mono";
import "./globals.css";
import { NavigationWrapper } from "@/components/navigation-wrapper";
import { QueryProvider } from "@/components/providers/query-provider";
import { ThemeProvider } from "@/components/providers/theme-provider";
import { SyntaxThemeProvider } from "@/components/providers/syntax-theme-provider";
import { FeatureFlagProvider } from "@/components/providers/feature-flag-provider";
import { Toaster } from "@/components/ui/sonner";
import { CommandPalette } from "@/components/command-palette";
import { env } from "@/lib/env";

export const metadata: Metadata = {
  title: "Ambient Code Platform",
  description:
    "ACP is an AI-native agentic-powered enterprise software development platform",
};

export const viewport = {
  width: "device-width",
  initialScale: 1,
};

// Force rebuild timestamp: 2025-11-20T16:38:00

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const wsBase = env.BACKEND_URL.replace(/^http:/, 'ws:').replace(/^https:/, 'wss:')
  const feedbackUrl = env.FEEDBACK_URL
  return (
    // suppressHydrationWarning is required for next-themes to prevent hydration mismatch
    // between server-rendered content and client-side theme application
    <html lang="en" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{__html: `if(!crypto.randomUUID){crypto.randomUUID=function(){return([1e7]+-1e3+-4e3+-8e3+-1e11).replace(/[018]/g,c=>(c^(crypto.getRandomValues(new Uint8Array(1))[0]&(15>>c/4))).toString(16))}}`}} />
        <meta name="backend-ws-base" content={wsBase} />
      </head>
      {/* suppressHydrationWarning is needed here as well since ThemeProvider modifies the class attribute */}
      <body className={`${GeistSans.variable} ${GeistMono.variable} font-sans min-h-screen flex flex-col`} suppressHydrationWarning>
        <ThemeProvider
          attribute="class"
          defaultTheme="system"
          enableSystem
          disableTransitionOnChange
        >
          <SyntaxThemeProvider />
          <FeatureFlagProvider>
            <QueryProvider>
              <NavigationWrapper feedbackUrl={feedbackUrl} />
              <main className="flex-1 bg-background overflow-auto">{children}</main>
              <CommandPalette />
              <Toaster />
            </QueryProvider>
          </FeatureFlagProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
