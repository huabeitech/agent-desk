import type { Metadata } from "next"
import { Geist, Geist_Mono } from "next/font/google"
import Script from "next/script"

import { ImageLightboxProvider } from "@/components/image-lightbox"
import { ConfirmProvider } from "@/components/confirm-provider"
import { SupportAuthProvider } from "@/app/(support)/support/_components/support-auth-provider"
import { ThemeProvider } from "@/components/theme-provider"
import { TooltipProvider } from "@/components/ui/tooltip"
import { Toaster } from "@/components/ui/sonner"
import { AppI18nProvider } from "@/i18n/provider"
import { themeInitializerScript } from "@/lib/theme"

import "./support.css"
import "md-editor-rt/lib/style.css"

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
})

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
})

export const metadata: Metadata = {
  title: "AgentDesk Support",
  description: "AgentDesk Support Center",
}

export default function SupportRootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en-US" className={`${geistSans.variable} ${geistMono.variable}`} suppressHydrationWarning>
      <body
        className="antialiased font-sans"
      >
        <Script id="support-theme-initializer" strategy="beforeInteractive">
          {themeInitializerScript}
        </Script>
        <AppI18nProvider>
          <ThemeProvider>
            <SupportAuthProvider>
              <ConfirmProvider>
                <ImageLightboxProvider>
                  <TooltipProvider>
                    {children}
                    <Toaster position="top-center" richColors />
                  </TooltipProvider>
                </ImageLightboxProvider>
              </ConfirmProvider>
            </SupportAuthProvider>
          </ThemeProvider>
        </AppI18nProvider>
      </body>
    </html>
  )
}
