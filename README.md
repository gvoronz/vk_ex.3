
## Полная инструкция для запуска проекта

### 1. Установка зависимостей

#### 1.1. Установка PostgreSQL

Если у вас еще нет PostgreSQL на вашем компьютере, следуйте этим инструкциям:

- **Для macOS**:
  - Если у вас еще нет `brew`, установите его с помощью команды:

    ```bash
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    ```

  - Установите PostgreSQL:

    ```bash
    brew install postgresql
    ```

  - Запустите службу PostgreSQL:

    ```bash
    brew services start postgresql
    ```

  - Чтобы проверить, что PostgreSQL работает, используйте:

    ```bash
    pg_isready
    ```

- **Для Linux**:
  - Используйте следующие команды:

    ```bash
    sudo apt update
    sudo apt install postgresql postgresql-contrib
    sudo service postgresql start
    ```

- **Для Windows**:
  - Загрузите и установите PostgreSQL с [официального сайта](https://www.postgresql.org/download/windows/).

#### 1.2. Установка Go

Если у вас еще нет Go, установите его следующим образом:

- Перейдите на сайт [Go Downloads](https://go.dev/dl/) и скачайте соответствующую версию для вашей операционной системы.
- Установите Go согласно инструкциям.

Чтобы проверить установку, выполните команду:

```bash
go version
```

#### 1.3. Установка Node.js и npm (для фронтенда)

Для работы с React-проектом необходимо установить **Node.js** и **npm**:

- Перейдите на сайт [Node.js](https://nodejs.org/) и скачайте установочный файл.
- После установки проверьте, что Node.js и npm установлены:

  ```bash
  node -v
  npm -v
  ```

### 2. Настройка базы данных

Создаем пользователя и базу данных для данного проекта:

1. Откройте консоль PostgreSQL:

   ```bash
   psql -U postgres
   ```

2. Создайте нового пользователя и базу данных:

   ```sql
   CREATE USER youruser WITH PASSWORD 'yourpassword';
   CREATE DATABASE yourdb;
   GRANT ALL PRIVILEGES ON DATABASE yourdb TO youruser;
   ```

3. Проверьте подключение к базе данных:

   ```bash
   psql -U youruser -d yourdb
   ```

### 3. Установка Docker (по желанию)

Если вы хотите запускать PostgreSQL через Docker:

1. Установите Docker с [официального сайта](https://www.docker.com/get-started).
2. Запустите контейнер с PostgreSQL:

   ```bash
   docker run --name some-postgres -e POSTGRES_PASSWORD=mysecretpassword -d postgres
   ```

   Где:
   - `some-postgres` — имя контейнера
   - `mysecretpassword` — пароль для пользователя `postgres`
   
3. Получите IP контейнера (если необходимо):

   ```bash
   docker inspect some-postgres | grep "IPAddress"
   ```

### 4. Настройка сервера на Go

Теперь перейдем к настройке серверной части на Go:

1. Создаем каталог для данного проекта (например, `ping-app`). В нашем случае это "backend"
2. В этом каталоге создаем файл `main.go` с содержимым:

```go
package main

import (
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
	"github.com/rs/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PingResult struct {
	ID        uint      `gorm:"primaryKey"`
	IPAddress string    `gorm:"not null"`
	PingTime  float64   `gorm:"not null"`
	LastSeen  time.Time `gorm:"not null"`
}

var db *gorm.DB

func initDB() {
	dsn := "host=localhost user=youruser password=yourpassword dbname=yourdb port=5432 sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	db.AutoMigrate(&PingResult{})
}

func getPingResults(c *gin.Context) {
	var results []PingResult
	db.Find(&results)
	c.JSON(http.StatusOK, results)
}

func pingContainers() {
	for {
		cmd := exec.Command("docker", "ps", "--format", "{{.ID}}")
		output, err := cmd.Output()
		if err != nil {
			log.Println("Failed to get container list:", err)
			continue
		}
		containerIDs := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, id := range containerIDs {
			cmd := exec.Command("docker", "inspect", "-f", "{{.NetworkSettings.IPAddress}}", id)
			ipOutput, err := cmd.Output()
			if err != nil {
				log.Println("Failed to get IP for container", id, ":", err)
				continue
			}
			ip := strings.TrimSpace(string(ipOutput))
			pingCmd := exec.Command("ping", "-c", "1", "-W", "1", ip)
			start := time.Now()
			if err := pingCmd.Run(); err == nil {
				pingTime := time.Since(start).Seconds() * 1000
				db.Create(&PingResult{IPAddress: ip, PingTime: pingTime, LastSeen: time.Now()})
			}
		}
		time.Sleep(30 * time.Second)
	}
}

func main() {
	initDB()
	go pingContainers()
	r := gin.Default()

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"},
	})
	r.Use(corsMiddleware)

	r.GET("/ping-results", getPingResults)
	r.Run(":8080")
}
```

### 5. Запуск сервера на Go

1. Выполните команду для запуска сервера:

   ```bash
   go run main.go
   ```

2. Если все настроено правильно, сервер будет доступен по адресу `http://localhost:8080`.

### 6. Настройка фронтенда (React)

Теперь давайте настроим фронтенд, который будет отображать результаты пинга:

1. Перейдите в каталог вашего проекта и выполните команду для создания нового React-приложения:

   ```bash
   npx create-react-app frontend
   ```

2. Перейдите в папку с фронтендом:

   ```bash
   cd frontend
   ```

3. Установите необходимые зависимости:

   ```bash
   npm install antd moment
   ```

4. В файле `src/App.js` используйте следующий код для отображения результатов пинга:

```js
import { useEffect, useState } from "react";
import { Table, Spin, Alert } from "antd";
import moment from "moment";

const fetchData = async () => {
  try {
    setLoading(true);
    setError("");  // Сброс ошибки перед новым запросом

    const response = await fetch("http://localhost:8080/ping-results");

    if (!response.ok) {
      const errorText = await response.text();  // Получаем текст ошибки от сервера
      throw new Error(`Ошибка при получении данных: ${response.status} ${errorText}`);
    }

    const result = await response.json();
    setData(result);
  } catch (error) {
    console.error("Ошибка загрузки:", error);  // Логируем ошибку в консоль
    setError(`Не удалось загрузить данные. ${error.message}`);  // Отображаем ошибку пользователю
  } finally {
    setLoading(false);  // Окончание загрузки
  }
};


const columns = [
  { title: "IP Address", dataIndex: "IPAddress", key: "ip" },
  { title: "Ping Time (ms)", dataIndex: "PingTime", key: "pingTime" },
  {
    title: "Last Seen",
    dataIndex: "LastSeen",
    key: "lastSeen",
    render: (text) => moment(text).format("YYYY-MM-DD HH:mm:ss"),
  },
];

export default function App() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    fetchData();  // Получаем данные сразу при загрузке компонента
    const interval = setInterval(fetchData, 30000);  // Периодический запрос данных каждые 30 секунд
    return () => clearInterval(interval);  // Очищаем интервал при размонтировании компонента
  }, []);

  return (
    <div className="p-4">
      <h2 className="text-xl font-bold mb-4">Результаты пинга</h2>

      {error && (
        <Alert message={error} type="error" showIcon className="mb-4" />
      )}
      {loading ? (
        <Spin size="large" className="block mx-auto mt-4" />
      ) : (
        <Table dataSource={data} columns={columns} rowKey="ID" />
      )}
    </div>
  );
}
```

5. Запускаем фронтенд:

   ```bash
   npm start
   ```

6. Приложение теперь будет доступно по адресу `http://localhost:3000` и будет отображать результаты пинга с сервера.

---

Теперь система полностью настроена для получения и отображения данных о пинге контейнеров.

Задачу считаю выполненной.
