import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./hooks/**/*.{ts,tsx}",
    "./lib/**/*.{ts,tsx}"
  ],
  theme: {
    extend: {
      colors: {
        ink: "#0f172a",
        panel: "#f8fafc",
        accent: "#0f766e",
        alert: "#b45309"
      },
      boxShadow: {
        panel: "0 10px 35px rgba(15, 23, 42, 0.08)"
      },
      fontFamily: {
        sans: ["IBM Plex Sans", "Avenir Next", "Segoe UI", "sans-serif"],
        mono: ["IBM Plex Mono", "SFMono-Regular", "monospace"]
      }
    }
  },
  plugins: []
};

export default config;
