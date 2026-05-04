"use client";

import { useState, useEffect } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { TooltipProvider } from "@/components/ui/tooltip";
import { getQueryClient } from "@/lib/api/query-client";
import { Theme, getStoredTheme, setStoredTheme, applyTheme } from "@/lib/theme/themes";

export function Providers({ children }: { children: React.ReactNode }) {
  const queryClient = getQueryClient();
  const [theme, setTheme] = useState<Theme>("dark");
  const [mswReady, setMswReady] = useState(false);

  useEffect(() => {
    const stored = getStoredTheme();
    setTheme(stored);
    applyTheme(stored);
  }, []);

  useEffect(() => {
    if (process.env.NODE_ENV === "development" && process.env.NEXT_PUBLIC_API_BASE_URL === undefined) {
      import("@/lib/mocks/browser").then(({ installMocks }) => {
        installMocks();
        setMswReady(true);
      });
    } else {
      setMswReady(true);
    }
  }, []);

  const changeTheme = (t: Theme) => {
    setTheme(t);
    setStoredTheme(t);
    applyTheme(t);
  };

  if (!mswReady) {
    return (
      <div className="flex h-screen items-center justify-center bg-background text-foreground">
        <span className="font-mono text-xs">[LOADING...]</span>
      </div>
    );
  }

  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={300}>
        <ThemeContext.Provider value={{ theme, setTheme: changeTheme }}>
          {children}
        </ThemeContext.Provider>
      </TooltipProvider>
    </QueryClientProvider>
  );
}

import { createContext, useContext } from "react";

type ThemeContextType = { theme: Theme; setTheme: (t: Theme) => void };
export const ThemeContext = createContext<ThemeContextType>({ theme: "dark", setTheme: () => {} });
export function useTheme() {
  return useContext(ThemeContext);
}
