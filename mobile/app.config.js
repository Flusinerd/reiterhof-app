// Lets local development point the app at another API without editing app.json:
// STALLFUNK_API_URL=http://10.0.2.2:8080 npx expo start
// On web the API is the origin of the page unless STALLFUNK_API_URL (or EXPO_PUBLIC_API_URL) is set
// when the app is built, e.g. STALLFUNK_API_URL=http://localhost:8080 npx expo export --platform web.
module.exports = ({ config }) => ({
  ...config,
  extra: {
    ...config.extra,
    apiUrl: process.env.STALLFUNK_API_URL || config.extra.apiUrl,
    apiUrlOverride: process.env.STALLFUNK_API_URL || "",
  },
});
