// Пакет main содержит реализацию хеш-функции ГОСТ Р 34.11-94
package main

import (
	"encoding/hex"    // Пакет для кодирования/декодирования шестнадцатеричных строк
	"fmt"             // Пакет для форматированного ввода-вывода
	"html/template"   // Пакет для работы с HTML шаблонами
	"io"              // Пакет для работы с операциями ввода-вывода
	"main/gost341194" // Импорт пакета с реализацией ГОСТ Р 34.11-94
	"net/http"        // Пакет для создания HTTP сервера
	"os"              // Пакет для работы с операционной системой
)

// Структура для хранения данных о результате хеширования
type HashResult struct {
	InputText string
	FileName  string
	Hash      string
	Error     string
}

// HTML шаблон для веб-интерфейса
const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>ГОСТ Р 34.11-94 Хеширование</title>
    <meta charset="utf-8">
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        .container {
            background-color: #f5f5f5;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .form-group {
            margin-bottom: 15px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        textarea {
            width: 100%;
            height: 100px;
            padding: 8px;
            box-sizing: border-box;
        }
        .result {
            margin-top: 20px;
            padding: 15px;
            background-color: #e8f5e9;
            border-radius: 4px;
            word-break: break-all;
        }
        .error {
            background-color: #ffebee;
        }
        button {
            background-color: #4CAF50;
            color: white;
            padding: 10px 15px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background-color: #45a049;
        }
        .tabs {
            display: flex;
            margin-bottom: 15px;
        }
        .tab {
            padding: 10px 15px;
            cursor: pointer;
            border: 1px solid #ddd;
            background-color: #f1f1f1;
            margin-right: 5px;
        }
        .tab.active {
            background-color: #fff;
            border-bottom: 1px solid #fff;
        }
        .tab-content {
            display: none;
        }
        .tab-content.active {
            display: block;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Генератор хеша ГОСТ Р 34.11-94</h1>
        
        <div class="tabs">
            <div class="tab active" onclick="openTab(event, 'text-tab')">Ввод текста</div>
            <div class="tab" onclick="openTab(event, 'file-tab')">Загрузка файла</div>
        </div>
        
        <div id="text-tab" class="tab-content active">
            <form action="/hash" method="post">
                <div class="form-group">
                    <label for="text">Введите текст для хеширования:</label>
                    <textarea id="text" name="text" required>{{.InputText}}</textarea>
                </div>
                <button type="submit">Хешировать</button>
            </form>
        </div>
        
        <div id="file-tab" class="tab-content">
            <form action="/hash-file" method="post" enctype="multipart/form-data">
                <div class="form-group">
                    <label for="file">Выберите файл для хеширования:</label>
                    <input type="file" id="file" name="file" required>
                </div>
                <button type="submit">Хешировать файл</button>
            </form>
        </div>
        
        {{if .Hash}}
        <div class="result">
            {{if .FileName}}
            <h3>Результат хеширования файла: {{.FileName}}</h3>
            {{else}}
            <h3>Результат хеширования текста:</h3>
            <p><strong>Исходный текст:</strong> {{.InputText}}</p>
            {{end}}
            <p><strong>ГОСТ Р 34.11-94 хеш:</strong> {{.Hash}}</p>
        </div>
        {{end}}
        
        {{if .Error}}
        <div class="result error">
            <h3>Ошибка:</h3>
            <p>{{.Error}}</p>
        </div>
        {{end}}
    </div>
    
    <script>
        function openTab(evt, tabName) {
            var i, tabcontent, tablinks;
            
            tabcontent = document.getElementsByClassName("tab-content");
            for (i = 0; i < tabcontent.length; i++) {
                tabcontent[i].className = tabcontent[i].className.replace(" active", "");
            }
            
            tablinks = document.getElementsByClassName("tab");
            for (i = 0; i < tablinks.length; i++) {
                tablinks[i].className = tablinks[i].className.replace(" active", "");
            }
            
            document.getElementById(tabName).className += " active";
            evt.currentTarget.className += " active";
        }
    </script>
</body>
</html>
`

// Функция для обработки главной страницы
func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		http.Error(w, "Ошибка шаблона: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, &HashResult{})
}

// Функция для хеширования текста
func hashTextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	text := r.FormValue("text")

	// Создаем хеш
	h := gost341194.New(gost341194.SboxDefault)
	h.Write([]byte(text))
	hash := h.Sum(nil)

	// Формируем результат
	result := &HashResult{
		InputText: text,
		Hash:      hex.EncodeToString(hash),
	}

	// Отображаем страницу с результатом
	tmpl, _ := template.New("index").Parse(htmlTemplate)
	tmpl.Execute(w, result)
}

// Функция для хеширования файла
func hashFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Получаем загруженный файл
	file, header, err := r.FormFile("file")
	if err != nil {
		renderError(w, "Не удалось получить файл: "+err.Error())
		return
	}
	defer file.Close()

	// Создаем хеш
	h := gost341194.New(gost341194.SboxDefault)

	// Копируем содержимое файла в хеш
	_, err = io.Copy(h, file)
	if err != nil {
		renderError(w, "Ошибка при чтении файла: "+err.Error())
		return
	}

	hash := h.Sum(nil)

	// Формируем результат
	result := &HashResult{
		FileName: header.Filename,
		Hash:     hex.EncodeToString(hash),
	}

	// Отображаем страницу с результатом
	tmpl, _ := template.New("index").Parse(htmlTemplate)
	tmpl.Execute(w, result)
}

// Функция для отображения ошибок
func renderError(w http.ResponseWriter, errMessage string) {
	result := &HashResult{
		Error: errMessage,
	}

	tmpl, _ := template.New("index").Parse(htmlTemplate)
	tmpl.Execute(w, result)
}

// computeFileHash вычисляет хеш для указанного файла
func computeFileHash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	h := gost341194.New(gost341194.SboxDefault)
	h.Write(data)
	hash := h.Sum(nil)

	return hex.EncodeToString(hash), nil
}

// main запускает веб-сервер или выполняет хеширование из командной строки
func main() {
	// Если указан путь к файлу в аргументах, вычисляем хеш файла
	if len(os.Args) > 1 {
		filePath := os.Args[1]
		hash, err := computeFileHash(filePath)
		if err != nil {
			fmt.Printf("Ошибка: %v\n", err)
		} else {
			fmt.Printf("ГОСТ Р 34.11-94 хеш для файла %s: %s\n", filePath, hash)
		}
		return
	}

	// Если аргументов нет, запускаем веб-сервер
	fmt.Println("Запуск веб-сервера на http://localhost:8080")

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/hash", hashTextHandler)
	http.HandleFunc("/hash-file", hashFileHandler)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("Ошибка запуска сервера: %v\n", err)
	}
}
