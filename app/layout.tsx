import "@/styles/globals.css";
import { Metadata, Viewport } from "next";
import { Inter } from "next/font/google";
import Link from "next/link";
import { NextUIProvider } from "@nextui-org/system";

import { siteConfig } from "@/config/site";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: {
    default: siteConfig.name,
    template: `%s - ${siteConfig.name}`,
  },
  description: siteConfig.description,
  icons: {
    icon: "/favicon.ico",
    shortcut: "/favicon-16x16.png",
    apple: "/apple-touch-icon.png",
  },
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "white" },
    { media: "(prefers-color-scheme: dark)", color: "black" },
  ],
  width: "device-width",
  initialScale: 1,
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html className="dark" lang="en">
      <head />
      <body
        className={
          "min-h-screen bg-background dark:bg-[#3C3D37] " + inter.className
        }
      >
        <NextUIProvider>
          <div className="relative flex flex-col min-h-screen">
            <nav className="w-full bg-[#181C14] shadow-md">
              <div className="container mx-auto px-4 py-3">
                <div className="flex justify-between items-center">
                  <span className="text-white text-xl font-semibold">
                    Video Queue
                  </span>
                  <div className="flex gap-4">
                    <Link className="text-gray-300 hover:text-white" href="/">
                      Moderation
                    </Link>
                    <Link
                      className="text-gray-300 hover:text-white"
                      href="/queue"
                    >
                      Queue Status
                    </Link>
                  </div>
                </div>
              </div>
            </nav>
            <main className="flex-grow container mx-auto px-4 py-8">
              {children}
            </main>
          </div>
        </NextUIProvider>
      </body>
    </html>
  );
}
