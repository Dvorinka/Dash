export type Theme = "light" | "dark" | "casaos";

const STORAGE_KEY = "dash-theme";

export function getStoredTheme(): Theme {
  if (typeof window === "undefined") return "dark";
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "casaos") return stored;
  return "dark";
}

export function setStoredTheme(theme: Theme) {
  if (typeof window === "undefined") return;
  localStorage.setItem(STORAGE_KEY, theme);
}

export function applyTheme(theme: Theme) {
  if (typeof document === "undefined") return;
  document.documentElement.setAttribute("data-theme", theme);
}

export const themeLabels: Record<Theme, string> = {
  light: "Light",
  dark: "Dark",
  casaos: "CasaOS",
};
