const fs = require("fs");
const path = require("path");

// Lets local development point the app at another API without editing app.json:
// STALLFUNK_API_URL=http://10.0.2.2:8080 npx expo start
// On web the API is the origin of the page unless STALLFUNK_API_URL (or EXPO_PUBLIC_API_URL) is set
// when the app is built, e.g. STALLFUNK_API_URL=http://localhost:8080 npx expo export --platform web.
//
// Push on Android needs the Firebase config of the operator's project (google-services.json
// from the Firebase console, git-ignored, see README). It is only wired in when the file
// exists, so web exports and builds without push still work.
const googleServicesFile = path.join(__dirname, "google-services.json");

module.exports = ({ config }) => ({
  ...config,
  android: {
    ...config.android,
    ...(fs.existsSync(googleServicesFile) ? { googleServicesFile: "./google-services.json" } : {}),
  },
  extra: {
    ...config.extra,
    apiUrl: process.env.STALLFUNK_API_URL || config.extra.apiUrl,
    apiUrlOverride: process.env.STALLFUNK_API_URL || "",
  },
});
