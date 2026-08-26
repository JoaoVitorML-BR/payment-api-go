"use client";
// fake_frontend_payment_test\frontend_test\app\layout.tsx
import { MantineProvider, createTheme } from "@mantine/core";

const theme = createTheme({
  primaryColor: "blue",
  defaultRadius: "md",
});

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <MantineProvider theme={theme}>
          {children}
        </MantineProvider>
      </body>
    </html>
  );
}
