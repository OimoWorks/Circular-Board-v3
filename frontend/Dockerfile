FROM ghcr.io/cirruslabs/flutter:3.24.0

WORKDIR /app

# 依存関係のインストール
COPY pubspec.yaml ./
RUN flutter pub get

# ソースコードをコピー
COPY . .

EXPOSE 3000

# Flutter Webの開発サーバーで起動
CMD ["flutter", "run", "-d", "web-server", \
     "--web-port=3000", \
     "--web-hostname=0.0.0.0", \
     "--dart-define=API_BASE_URL=http://localhost:8080"]
