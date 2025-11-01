#!/bin/bash
echo "🚀 Voyager Proxy Installer"
echo "=========================="

# Убираем карантин с приложения
echo "🔓 Unlocking application..."
xattr -dr com.apple.quarantine VoyagerProxyNetwork.app

echo "✅ Installation complete!"
echo "📍 You can now open VoyagerProxyNetwork.app normally"
echo "💡 If you still see warnings, right-click and select 'Open'"

# Открываем папку с приложением
open .