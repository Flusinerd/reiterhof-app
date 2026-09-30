// Lets local development point the app at another API without editing app.json:
// STALLFUNK_API_URL=http://10.0.2.2:8080 npx expo start
module.exports = ({ config }) => ({
  ...config,
  extra: { ...config.extra, apiUrl: process.env.STALLFUNK_API_URL || config.extra.apiUrl },
});
