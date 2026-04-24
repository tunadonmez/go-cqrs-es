import type { Metadata } from "next";
import type { ReactNode } from "react";

import { AppShell } from "@/components/layout/app-shell";

import "./globals.css";
import { Providers } from "./providers";

export const metadata: Metadata = {
  title: "CQRS Admin Console",
  description: "Technical admin console for the wallet CQRS and event sourcing demo."
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body>
        <Providers>
          <AppShell>{children}</AppShell>
        </Providers>
      </body>
    </html>
  );
}
