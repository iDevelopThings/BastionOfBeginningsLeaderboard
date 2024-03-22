/** @type {import('tailwindcss').Config} */
module.exports = {
    content: ["./src/**/*.{html,js,gohtml}"],
    darkMode: 'media',

    theme: {
        extend: {},
    },
    plugins: [
        require('@tailwindcss/typography'),
    ],
}

