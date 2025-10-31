# Voyager Proxy Network

![Voyager Logo](assets/64X64.png)

Современное, удобное VPN-приложение, разработанное на языке Go с использованием фреймворка Fyne UI. Voyager Proxy Network предоставляет безопасный способ управления VPN-соединениями через простой и минималистичный интерфейс. Проект реализует сложное решение для прокси-сетей с акцентом на безопасность, производительность и удобство использования.

## 🛠 Технологический стек

### Бэкенд
- **Go 1.25.1** - Основной язык программирования
- **XRay Core** - Реализация базового прокси-протокола
  - Управление пользовательскими конфигурациями
  - Поддержка протоколов: VLESS, VMess, Trojan, Shadowsocks
  - Маршрутизация трафика и управление политиками

### Фронтенд и UI
- **Fyne v2.7.0** - Кроссплатформенный GUI фреймворк
  - Реализация пользовательских виджетов
  - Адаптивная система макетов
  - Поддержка светлой и темной тем
- **Интеграция с системным треем** - Функциональность системного трея для десктопных сред

### Сетевые возможности
- **Поддержка протоколов TCP/UDP**
- **Настраиваемое разрешение DNS**
- **Управление IP-адресами**
- **Маршрутизация трафика**

### Хранение и конфигурация
- **JSON-конфигурация**
- **Система локального хранения**
- **Функции импорта/экспорта конфигураций**

### Инструменты разработки
- **Make** - Автоматизация сборки
- **Go Modules** - Управление зависимостями
- **Git** - Система контроля версий

## ✨ Key Features

### Connection Management
- 🔒 Secure VPN connection handling with multiple protocol support
- 🔄 Automatic reconnection and connection health monitoring
- 📊 Real-time connection statistics and diagnostics
- 🌡️ Connection quality indicators and latency monitoring

### User Interface
- 🎨 Modern, minimal, and intuitive interface design
- 🌓 Dark/Light theme support with custom styling
- 🖥️ Cross-platform desktop application (primary support for macOS)
- 🔔 System tray integration with status indicators
- 📱 Responsive layout for different window sizes

### Configuration & Settings
- ⚙️ Advanced configuration management system
- 📥 Import/Export configuration functionality
- � Custom proxy settings configuration
- 📝 Connection profiles management
- 🔍 Configuration validation and error checking

### Security Features
- 🛡️ Multiple security protocol support (VLESS, VMess, Trojan, Shadowsocks)
- 🔐 Encrypted connections and secure data transmission
- �️ IP address leak prevention
- 🚦 Traffic routing policies
- 🌐 DNS leak protection

### System Integration
- 💻 Native system tray integration
- 🚀 Lightweight system resource usage
- 📊 System-wide proxy configuration
- 🔄 Automatic updates support
- 📡 Network interface management

## Требования

- Go версии 1.25.1 или выше
- Фреймворк Fyne UI
- Зависимости XRay core

## Установка

1. Клонировать репозиторий:
```bash
git clone https://github.com/Cherylem/Voyager_Proxy_Networ.git
cd Voyager_Proxy_Networ
```

2. Установить зависимости:
```bash
go mod download
```

3. Собрать приложение:
```bash
make build
```

## Структура проекта

```
├── assets/           # Иконки и изображения приложения
├── configurations/   # Файлы конфигурации
├── core/            # Интеграция с XRay core
├── models/          # Модели данных
├── services/        # Сервисы бизнес-логики
├── themes/          # UI темы
└── ui/              # Компоненты пользовательского интерфейса
```

## Использование

1. Запустите приложение:
```bash
./vpn-client
```

2. Приложение появится в системном трее
3. Используйте иконку в трее для управления VPN-соединениями
4. Настройте параметры подключения через пользовательский интерфейс

## 🔧 Разработка

### Обзор архитектуры

Проект следует модульной архитектуре с четким разделением ответственности:

```
core/
├── xray_config.go   # Управление конфигурацией XRay
└── xray_manager.go  # Основной функционал VPN

models/
└── connection.go    # Модели данных для соединений

services/
├── checkIP.go      # Сервис проверки IP
├── parser.go       # Парсер конфигураций
└── storage.go      # Управление локальным хранилищем

ui/
├── app.go          # Основная логика приложения
├── components.go   # UI компоненты
└── tray.go         # Интеграция с системным треем
```

### Ключевые компоненты

1. **Модуль Core**
   - Генерация и управление конфигурацией XRay
   - Управление жизненным циклом соединения
   - Реализация протоколов
   - Маршрутизация трафика

2. **Модуль Services**
   - Парсинг и валидация конфигураций
   - Проверка IP-адресов
   - Управление локальным хранилищем
   - Сохранение настроек

3. **Модуль UI**
   - Управление главным окном приложения
   - Пользовательские UI компоненты
   - Интеграция с системным треем
   - Управление темами

4. **Модуль Models**
   - Структуры данных для соединений
   - Модели конфигураций
   - Модели мониторинга состояния

### Инструменты разработки

- [Fyne](https://fyne.io/) - Современный и мощный GUI фреймворк
- Go modules - Управление зависимостями с версионированием
- XRay - Реализация основного VPN протокола
- Make - Автоматизация сборки и выполнения задач

## 🏗 Сборка из исходного кода

### Предварительные требования
```bash
# Установить Go 1.25.1 или выше
go version

# Установить необходимые системные зависимости
brew install make     # Для macOS
```

### Шаги сборки
```bash
# Сборка приложения с оптимизациями
go build -ldflags="-s -w" -o vpn-client

# Или просто использовать make
make build

# Запуск приложения
./vpn-client
```

### Сборка для разработки
```bash
# Запуск напрямую через Go
go run main.go

# Сборка с отладочной информацией
go build -o vpn-client-debug
```

### Флаги сборки
- `-ldflags="-s -w"` - Уменьшает размер бинарного файла путем удаления отладочной информации
- `-o vpn-client` - Указывает имя выходного бинарного файла

## Как внести свой вклад

1. Сделайте форк репозитория
2. Создайте ветку для новой функции (`git checkout -b feature/amazing-feature`)
3. Зафиксируйте изменения (`git commit -m 'Добавлена новая функция'`)
4. Отправьте изменения в репозиторий (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

## Лицензия

Этот проект лицензирован под MIT License - подробности см. в файле LICENSE.

## Контакты

Ссылка на проект: [https://github.com/Cherylem/Voyager_Proxy_Networ](https://github.com/Cherylem/Voyager_Proxy_Networ)

---

Создано с ❤️ разработчиком Cherylem
