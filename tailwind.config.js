/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./ui/templates/*.templ"],
  theme: {
    extend: {
      colors: {
        "chatopia-1": "#20B43C",
        "chatopia-2": "#01161E",
        "chatopia-3": "#124559",
        "chatopia-4": "#598392",
        "chatopia-5": "#EFF6E0",
      },
    },
  },
  plugins: [],
}

