APP_NAME=VoyagerProxyNetwork
ICON_DIR=AppIcon/dark_icon/apple-devices/AppIcon.appiconset/

# Основная цель
app: directory info build move make_icon
	@echo "✅ Building app success"
build:
	MACOSX_DEPLOYMENT_TARGET=13.0 \
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
	go build -ldflags="-s -w" -o VoyagerCore .

move:
	@echo "📦 Moving files to app bundle..."
	mv VoyagerCore $(APP_NAME).app/Contents/MacOS/
	mv info.plist $(APP_NAME).app/Contents/

	@# Копируем содержимое папки assets
	@if [ -d "assets" ]; then \
		cp -r assets/* $(APP_NAME).app/Contents/Resources/assets/ 2>/dev/null || echo "⚠️ Some assets not copied"; \
		echo "✅ Copied assets"; \
	else \
		echo "⚠️ assets folder not found"; \
	fi

directory:
	@echo "📁 Creating app directory structure..."
	@# Используем -p везде чтобы избежать ошибок если папки уже существуют
	mkdir -p $(APP_NAME).app/Contents/Resources/configurations
	mkdir -p $(APP_NAME).app/Contents/MacOS
	mkdir -p $(APP_NAME).app/Contents/Resources/assets

	@# Создаем config.json если его нет
	touch $(APP_NAME).app/Contents/Resources/configurations/config.json; \
	echo "📄 Created empty config.json"; \

make_icon:
	@echo "🎨 Creating app icon..."
	@if [ -d "$(ICON_DIR)" ]; then \
		mkdir -p myicon.iconset; \
		echo "📁 Copying icon files..."; \
		cp "$(ICON_DIR)"icon-mac-16x16.png myicon.iconset/icon_16x16.png 2>/dev/null || echo "⚠️ 16x16 not found"; \
		cp "$(ICON_DIR)"icon-mac-32x32.png myicon.iconset/icon_32x32.png 2>/dev/null || echo "⚠️ 32x32 not found"; \
		cp "$(ICON_DIR)"icon-mac-128x128.png myicon.iconset/icon_128x128.png 2>/dev/null || echo "⚠️ 128x128 not found"; \
		cp "$(ICON_DIR)"icon-mac-256x256.png myicon.iconset/icon_256x256.png 2>/dev/null || echo "⚠️ 256x256 not found"; \
		cp "$(ICON_DIR)"icon-mac-512x512.png myicon.iconset/icon_512x512.png 2>/dev/null || echo "⚠️ 512x512 not found"; \
		cp "$(ICON_DIR)"icon-mac-16x16@2x.png myicon.iconset/icon_16x16@2x.png 2>/dev/null || echo "⚠️ 16x16@2x not found"; \
		cp "$(ICON_DIR)"icon-mac-32x32@2x.png myicon.iconset/icon_32x32@2x.png 2>/dev/null || echo "⚠️ 32x32@2x not found"; \
		cp "$(ICON_DIR)"icon-mac-128x128@2x.png myicon.iconset/icon_128x128@2x.png 2>/dev/null || echo "⚠️ 128x128@2x not found"; \
		cp "$(ICON_DIR)"icon-mac-256x256@2x.png myicon.iconset/icon_256x256@2x.png 2>/dev/null || echo "⚠️ 256x256@2x not found"; \
		cp "$(ICON_DIR)"icon-mac-512x512@2x.png myicon.iconset/icon_512x512@2x.png 2>/dev/null || echo "⚠️ 512x512@2x not found"; \
		\
		iconutil -c icns myicon.iconset; \
		mv myicon.icns $(APP_NAME).app/Contents/Resources/AppIcon.icns; \
		rm -rf myicon.iconset; \
		echo "✅ App icon created"; \
	else \
		echo "❌ Icon directory not found: $(ICON_DIR)"; \
	fi

info:
	@echo "<?xml version=\"1.0\" encoding=\"UTF-8\"?>" > info.plist
	@echo "<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">" >> info.plist
	@echo "<plist version=\"1.0\">" >> info.plist
	@echo "<dict>" >> info.plist
	@echo "    <key>CFBundleExecutable</key>" >> info.plist
	@echo "    <string>VoyagerCore</string>" >> info.plist
	@echo "    <key>CFBundleIconFile</key>" >> info.plist
	@echo "    <string>AppIcon.icns</string>" >> info.plist
	@echo "    <key>CFBundleIdentifier</key>" >> info.plist
	@echo "    <string>com.voyager.proxy</string>" >> info.plist
	@echo "    <key>CFBundleName</key>" >> info.plist
	@echo "    <string>VoyagerProxyNetwork</string>" >> info.plist
	@echo "    <key>CFBundlePackageType</key>" >> info.plist
	@echo "    <string>APPL</string>" >> info.plist
	@echo "    <key>CFBundleVersion</key>" >> info.plist
	@echo "    <string>1.0.0</string>" >> info.plist
	@echo "    <key>CFBundleShortVersionString</key>" >> info.plist
	@echo "    <string>1.0</string>" >> info.plist
	@echo "    <key>LSMinimumSystemVersion</key>" >> info.plist
	@echo "    <string>13.0</string>" >> info.plist
	@echo "    <key>NSHighResolutionCapable</key>" >> info.plist
	@echo "    <true/>" >> info.plist
	@echo "    <key>LSUIElement</key>" >> info.plist
	@echo "    <true/>" >> info.plist
	@echo "</dict>" >> info.plist
	@echo "</plist>" >> info.plist

# Создание DMG
create-dmg: app
	@echo "📀 Creating DMG installer..."
	@rm -rf $(APP_NAME).dmg dmg_temp
	@mkdir -p dmg_temp
	cp -r $(APP_NAME).app dmg_temp/
	cp package_for_dmg/Install.command dmg_temp/
	ln -s /Applications dmg_temp/
	hdiutil create -volname "$(APP_NAME)" -srcfolder dmg_temp -ov -format UDZO "$(APP_NAME).dmg"
	rm -rf dmg_temp
	@echo "✅ $(APP_NAME).dmg created!"

unlock-app:
	@echo "🔓 Removing quarantine attribute..."
	xattr -dr com.apple.quarantine $(APP_NAME).app
	@echo "✅ App unlocked - can be opened normally"

# Полная сборка + разблокировка
distribute: app create-dmg unlock-app
	@echo "🎉 Distribution package ready!"
	@echo "📦 Users can now open the app normally"

clean:
	rm -rf $(APP_NAME).app $(APP_NAME).dmg VoyagerCore info.plist myicon.iconset

.PHONY: app build move directory make_icon info create-dmg clean