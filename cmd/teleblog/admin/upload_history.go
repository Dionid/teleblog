package admin

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Dionid/teleblog/cmd/teleblog/features"
	"github.com/Dionid/teleblog/libs/file"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/rest"
)

// InitUploadHistoryUI initializes the upload history admin UI routes
func InitUploadHistoryUI(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// Add the upload history page
		e.Router.GET("/_/upload-history", func(c echo.Context) error {
			html := `
				<!DOCTYPE html>
				<html lang="ru">
				<head>
					<title>Загрузка истории Telegram</title>
					<meta charset="utf-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&display=swap">
					<style>
						:root {
							--primary-color: #000000;
							--primary-hover: #333333;
							--success-color: #0070f3;
							--error-color: #ff0000;
							--bg-color: #fafafa;
							--card-bg: #ffffff;
							--text-color: #000000;
							--text-secondary: #666666;
							--border-color: #eaeaea;
							--border-hover: #000000;
						}

						* {
							margin: 0;
							padding: 0;
							box-sizing: border-box;
						}

						body {
							font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
							background-color: var(--bg-color);
							color: var(--text-color);
							line-height: 1.5;
							-webkit-font-smoothing: antialiased;
						}

						.container {
							max-width: 600px;
							margin: 80px auto;
							padding: 0 20px;
						}

						.card {
							background: var(--card-bg);
							border-radius: 5px;
							border: 1px solid var(--border-color);
							padding: 40px;
							margin-bottom: 24px;
							transition: border-color 0.15s ease;
						}

						.card:hover {
							border-color: var(--border-hover);
						}

						h1 {
							font-size: 28px;
							font-weight: 600;
							color: var(--text-color);
							margin-bottom: 12px;
							letter-spacing: -0.02em;
						}

						.subtitle {
							color: var(--text-secondary);
							font-size: 15px;
							margin-bottom: 32px;
						}

						.form-group {
							margin-bottom: 32px;
						}

						label {
							display: block;
							font-weight: 500;
							margin-bottom: 8px;
							color: var(--text-color);
							font-size: 14px;
						}

						.file-input-wrapper {
							position: relative;
							width: 100%;
							height: 140px;
							border: 1px dashed var(--border-color);
							border-radius: 5px;
							display: flex;
							align-items: center;
							justify-content: center;
							cursor: pointer;
							transition: all 0.15s ease;
						}

						.file-input-wrapper:hover {
							border-color: var(--border-hover);
							background-color: rgba(0, 0, 0, 0.02);
						}

						.file-input-wrapper.dragging {
							border-color: var(--border-hover);
							background-color: rgba(0, 0, 0, 0.02);
						}

						.file-input {
							position: absolute;
							width: 100%;
							height: 100%;
							opacity: 0;
							cursor: pointer;
						}

						.file-input-text {
							text-align: center;
							color: var(--text-secondary);
						}

						.file-input-text svg {
							margin-bottom: 12px;
							color: var(--text-color);
						}

						.file-input-text span {
							color: var(--text-color);
							font-weight: 500;
							text-decoration: underline;
						}

						.selected-file {
							display: none;
							margin-top: 12px;
							padding: 12px;
							background-color: var(--bg-color);
							font-size: 14px;
							color: var(--text-secondary);
						}

						button {
							background: var(--primary-color);
							color: white;
							border: none;
							padding: 12px 32px;
							border-radius: 5px;
							font-weight: 500;
							font-size: 14px;
							cursor: pointer;
							width: 100%;
							transition: all 0.15s ease;
							height: 48px;
							text-transform: uppercase;
							letter-spacing: 0.05em;
						}

						button:hover {
							background: var(--primary-hover);
						}

						button:disabled {
							opacity: 0.5;
							cursor: not-allowed;
						}

						.alert {
							padding: 16px;
							margin-top: 24px;
							border-radius: 5px;
							font-size: 14px;
							display: flex;
							align-items: flex-start;
							gap: 12px;
							border: 1px solid var(--border-color);
						}

						.alert svg {
							flex-shrink: 0;
							width: 16px;
							height: 16px;
							margin-top: 2px;
						}

						.alert-success {
							background: var(--bg-color);
							color: var(--text-color);
							border-left: 4px solid var(--success-color);
						}

						.alert-error {
							background: var(--bg-color);
							color: var(--text-color);
							border-left: 4px solid var(--error-color);
						}

						.loading {
							display: none;
							align-items: center;
							justify-content: center;
							gap: 8px;
							color: white;
						}

						.loading-spinner {
							width: 16px;
							height: 16px;
							border: 2px solid rgba(255, 255, 255, 0.3);
							border-top-color: white;
							border-radius: 50%;
							animation: spin 0.6s linear infinite;
						}

						@keyframes spin {
							to { transform: rotate(360deg); }
						}

						@media (max-width: 640px) {
							.container {
								margin: 40px auto;
							}
							.card {
								padding: 24px;
							}
							button {
								height: 44px;
							}
						}
					</style>
				</head>
				<body>
					<div class="container">
						<div class="card">
							<h1>Загрузка истории Telegram</h1>
							<p class="subtitle">Создайте и загрузите сюда Zip файл, с выгрузкой истории чата из Telegram (в формате JSON).</p>
							<form id="uploadForm" enctype="multipart/form-data">
								<div class="form-group">
									<label for="historyFile">Файл истории (ZIP)</label>
									<div class="file-input-wrapper" id="dropZone">
										<input type="file" id="historyFile" name="historyFile" accept=".zip" class="file-input" required>
										<div class="file-input-text">
											<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
												<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
												<polyline points="17 8 12 3 7 8"/>
												<line x1="12" y1="3" x2="12" y2="15"/>
											</svg>
											<p>Перетащите zip файл сюда или <span>выберите файл</span></p>
										</div>
									</div>
									<div class="selected-file" id="selectedFile"></div>
								</div>
								<button type="submit">
									<span class="button-text">Загрузить историю</span>
									<span class="loading">
										<div class="loading-spinner"></div>
										Загрузка...
									</span>
								</button>
							</form>
						</div>
						<div id="result"></div>
					</div>

					<script>
						// Get admin token from localStorage
						const token = JSON.parse(localStorage.getItem('pb_admin_auth')).token;

						const form = document.getElementById('uploadForm');
						const dropZone = document.getElementById('dropZone');
						const fileInput = document.getElementById('historyFile');
						const selectedFile = document.getElementById('selectedFile');
						const submitButton = form.querySelector('button');
						const buttonText = form.querySelector('.button-text');
						const loadingIndicator = form.querySelector('.loading');

						['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
							dropZone.addEventListener(eventName, preventDefaults, false);
							document.body.addEventListener(eventName, preventDefaults, false);
						});

						['dragenter', 'dragover'].forEach(eventName => {
							dropZone.addEventListener(eventName, highlight, false);
						});

						['dragleave', 'drop'].forEach(eventName => {
							dropZone.addEventListener(eventName, unhighlight, false);
						});

						dropZone.addEventListener('drop', handleDrop, false);

						function preventDefaults (e) {
							e.preventDefault();
							e.stopPropagation();
						}

						function highlight(e) {
							dropZone.classList.add('dragging');
						}

						function unhighlight(e) {
							dropZone.classList.remove('dragging');
						}

						function handleDrop(e) {
							const dt = e.dataTransfer;
							const files = dt.files;
							fileInput.files = files;
							updateFileName();
						}

						fileInput.addEventListener('change', updateFileName);

						function updateFileName() {
							const file = fileInput.files[0];
							if (file) {
								if (!file.name.toLowerCase().endsWith('.zip')) {
									selectedFile.style.display = 'block';
									selectedFile.textContent = 'Ошибка: Разрешены только ZIP файлы';
									selectedFile.style.color = 'var(--error-color)';
									submitButton.disabled = true;
									return;
								}
								selectedFile.style.display = 'block';
								selectedFile.textContent = file.name;
								selectedFile.style.color = 'var(--text-secondary)';
								submitButton.disabled = false;
							} else {
								selectedFile.style.display = 'none';
								submitButton.disabled = true;
							}
						}

						form.addEventListener('submit', async (e) => {
							e.preventDefault();
							const form = new FormData();
							const fileInput = document.getElementById('historyFile');
							form.append('historyFile', fileInput.files[0]);

							// Show loading state
							buttonText.style.display = 'none';
							loadingIndicator.style.display = 'flex';
							submitButton.disabled = true;

							try {
								const response = await fetch('/_/upload-history', {
									method: 'POST',
									body: form,
									headers: {
										'Authorization': token
									}
								});

								const data = await response.json();
								const resultDiv = document.getElementById('result');

								if (response.ok) {
									resultDiv.innerHTML = '<div class="alert alert-success"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd" /></svg>' + data.message + '</div>';
									fileInput.value = '';
									selectedFile.style.display = 'none';
								} else {
									resultDiv.innerHTML = '<div class="alert alert-error"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-8-5a.75.75 0 01.75.75v4.5a.75.75 0 01-1.5 0v-4.5A.75.75 0 0110 5zm0 10a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" /></svg>' + data.error + '</div>';
								}
							} catch (error) {
								document.getElementById('result').innerHTML = '<div class="alert alert-error"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-8-5a.75.75 0 01.75.75v4.5a.75.75 0 01-1.5 0v-4.5A.75.75 0 0110 5zm0 10a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" /></svg>Ошибка загрузки: ' + error.message + '</div>';
							} finally {
								// Reset loading state
								buttonText.style.display = 'block';
								loadingIndicator.style.display = 'none';
								submitButton.disabled = false;
							}
						});

						// Initial button state
						submitButton.disabled = !fileInput.files.length;
					</script>
				</body>
				</html>
			`
			return c.HTML(http.StatusOK, html)
		})

		// Add the upload history API endpoint
		e.Router.POST("/_/upload-history", func(c echo.Context) error {
			// Get the uploaded file
			files, err := rest.FindUploadedFiles(c.Request(), "historyFile")
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("Failed to get uploaded file: %v", err),
				})
			}

			if len(files) == 0 {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "No file was uploaded",
				})
			}

			uploadedFile := files[0]

			// Check if the file is a zip file
			if ext := filepath.Ext(uploadedFile.Name); ext != ".zip" {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Only zip files are allowed",
				})
			}

			// Read the file content
			reader, err := uploadedFile.Reader.Open()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": fmt.Sprintf("Failed to read file: %v", err),
				})
			}
			defer reader.Close()

			// Read the entire file into memory
			fileBytes, err := io.ReadAll(reader)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": fmt.Sprintf("Failed to read zip file: %v", err),
				})
			}

			// Create a bytes reader which implements io.ReaderAt
			zipReader, err := zip.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("Failed to read zip file: %v", err),
				})
			}

			// Unzip the zip file
			folderPathPrefix := "extracted-" + time.Now().Format("20060102150405")
			err = file.Unzip(zipReader, folderPathPrefix)
			if err != nil {
				log.Fatal(err)
			}
			defer os.RemoveAll(folderPathPrefix)

			// Upload the history
			if err := features.UploadHistory(app, folderPathPrefix); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": fmt.Sprintf("Failed to upload history: %v", err),
				})
			}

			return c.JSON(http.StatusOK, map[string]string{
				"message": "History uploaded successfully",
			})
		}, apis.RequireAdminAuth())

		return nil
	})
}
