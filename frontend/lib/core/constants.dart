// API_BASE_URL は --dart-define=API_BASE_URL=http://... で上書き可能
const String apiBaseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://localhost:8080',
);

const String accessTokenKey = 'access_token';
const String refreshTokenKey = 'refresh_token';
