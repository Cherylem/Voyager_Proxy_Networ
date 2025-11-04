#!/bin/bash

APP_NAME="VoyagerProxyNetwork.app"
DEST_DIR="/Applications"

# Определяем абсолютный путь к каталогу, где лежит скрипт
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_SOURCE="$SCRIPT_DIR/$APP_NAME"
APP_DEST="$DEST_DIR/$APP_NAME"

echo "🚀 Добро пожаловать в установку $APP_NAME!"
echo ""

# Проверяем наличие приложения рядом со скриптом
if [ ! -d "$APP_SOURCE" ]; then
    echo "❌ Ошибка: не найдено приложение $APP_NAME рядом со скриптом."
    echo "Проверьте, что $APP_NAME и Install.command лежат в одной папке."
    exit 1
fi

# Проверяем, установлено ли уже приложение
if [ -d "$APP_DEST" ]; then
    echo "⚠️  Приложение уже установлено в /Applications."
    read -p "Хотите заменить его? (y/n): " answer
    if [[ "$answer" =~ ^[Yy]$ ]]; then
        echo "🔁 Заменяем существующую версию..."
        sudo rm -rf "$APP_DEST"
    else
        echo "⏩ Пропускаем копирование. Запускаем установленное приложение..."
        open -a "$APP_DEST"
        exit 0
    fi
fi

# Копируем приложение в /Applications
echo "📦 Копирование $APP_NAME в /Applications..."
sudo cp -R "$APP_SOURCE" "$DEST_DIR"

# Проверяем успех
if [ ! -d "$APP_DEST" ]; then
    echo "❌ Ошибка: не удалось скопировать приложение. Проверьте права."
    exit 1
fi

# Снимаем quarantine (Gatekeeper)
xattr -dr com.apple.quarantine "$APP_DEST" 2>/dev/null

# Запускаем приложение
echo "🚀 Запуск приложения..."
open -a "$APP_DEST"

echo ""
echo "✅ Установка завершена! Приложение добавлено в /Applications и запущено."
echo "✨ Спасибо, что выбрали Voyager Proxy Network!"
echo ""

exit 0