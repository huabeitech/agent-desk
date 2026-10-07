export const THEME_STORAGE_KEY = "theme"

export const themeInitializerScript = `
try {
  var storedTheme = window.localStorage.getItem("theme");
  var theme = storedTheme === "light" || storedTheme === "dark" || storedTheme === "system" ? storedTheme : "system";
  var resolvedTheme = theme === "system"
    ? (window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light")
    : theme;
  document.documentElement.classList.remove("light", "dark");
  document.documentElement.classList.add(resolvedTheme);
  document.documentElement.style.colorScheme = resolvedTheme;
} catch (_) {}
`
