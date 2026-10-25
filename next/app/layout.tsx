import { Geist } from "next/font/google";
import type { Metadata, Viewport } from "next";
import "./globals.css";
import { ErrorTracker } from "@/components/error-tracker";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import SessionExtender from "@/features/session-extender";

const geist = Geist({ subsets: ["latin"], variable: "--font-geist" });

export const metadata: Metadata = {
  title: "Axion ERP",
  description: "Enterprise Resource Planning & Business Management System",
  generator: "v0.app",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
  colorScheme: "light dark",
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#f2f5f8" },
    { media: "(prefers-color-scheme: dark)", color: "#202735" },
  ],
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="id" className="bg-background" suppressHydrationWarning>
      <body
        className={`${geist.variable} antialiased`}
        suppressHydrationWarning
      >
        <TooltipProvider>
          <ErrorTracker />
          {children}
          <Toaster richColors position="top-center" />
          <SessionExtender />
        </TooltipProvider>
      </body>
    </html>
  );
}
