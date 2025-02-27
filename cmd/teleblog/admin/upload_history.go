package admin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Dionid/teleblog/cmd/teleblog/features"
	"github.com/Dionid/teleblog/libs/teleblog"
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
				<html>
				<head>
					<title>Upload Telegram Chat History</title>
					<meta charset="utf-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<style>
						body {
							font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
							max-width: 800px;
							margin: 0 auto;
							padding: 20px;
						}
						.form-group {
							margin-bottom: 20px;
						}
						label {
							display: block;
							margin-bottom: 5px;
							font-weight: bold;
						}
						input[type="file"] {
							display: block;
							width: 100%;
							padding: 10px;
							border: 1px solid #ddd;
							border-radius: 4px;
						}
						button {
							background: #16a34a;
							color: white;
							border: none;
							padding: 10px 20px;
							border-radius: 4px;
							cursor: pointer;
						}
						button:hover {
							background: #15803d;
						}
						.alert {
							padding: 15px;
							margin-bottom: 20px;
							border-radius: 4px;
						}
						.alert-success {
							background: #dcfce7;
							color: #166534;
							border: 1px solid #bbf7d0;
						}
						.alert-error {
							background: #fee2e2;
							color: #991b1b;
							border: 1px solid #fecaca;
						}
					</style>
				</head>
				<body>
					<h1>Upload Telegram Chat History</h1>
					<p>Upload a JSON file exported from Telegram to import chat history.</p>
					<form id="uploadForm" enctype="multipart/form-data">
						<div class="form-group">
							<label for="historyFile">History File (JSON)</label>
							<input type="file" id="historyFile" name="historyFile" accept=".json" required>
						</div>
						<button type="submit">Upload History</button>
					</form>
					<div id="result"></div>

					<script>
						document.getElementById('uploadForm').addEventListener('submit', async (e) => {
							e.preventDefault();
							const form = new FormData();
							const fileInput = document.getElementById('historyFile');
							form.append('historyFile', fileInput.files[0]);

							try {
								const response = await fetch('/_/upload-history', {
									method: 'POST',
									body: form,
									headers: {
										'Authorization': pb.authStore.token
									}
								});

								const data = await response.json();
								const resultDiv = document.getElementById('result');

								if (response.ok) {
									resultDiv.innerHTML = '<div class="alert alert-success">' + data.message + '</div>';
									fileInput.value = '';
								} else {
									resultDiv.innerHTML = '<div class="alert alert-error">' + data.error + '</div>';
								}
							} catch (error) {
								document.getElementById('result').innerHTML = '<div class="alert alert-error">Upload failed: ' + error.message + '</div>';
							}
						});
					</script>
				</body>
				</html>
			`
			return c.HTML(http.StatusOK, html)
		}, apis.RequireAdminAuth())

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

			file := files[0]

			// Read the file content
			reader, err := file.Reader.Open()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": fmt.Sprintf("Failed to read file: %v", err),
				})
			}
			defer reader.Close()

			// Parse the JSON content
			var history teleblog.History
			if err := json.NewDecoder(reader).Decode(&history); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("Invalid JSON format: %v", err),
				})
			}

			// Upload the history
			if err := features.UploadHistory(app, history); err != nil {
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
