// Убедитесь, что этот код загружается после DOM
document.addEventListener('DOMContentLoaded', function() {
    console.log('DOM loaded, initializing ImageProcessor...');
    
    // Инициализация приложения
    const imageProcessor = new ImageProcessor();
    window.imageProcessor = imageProcessor;
});

class ImageProcessor {
    constructor() {
        this.images = new Map();
        this.originalImages = new Map();
        console.log('ImageProcessor constructor called');
        
        // Ждем полной загрузки DOM перед настройкой
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => {
                this.setupDragAndDrop();
                this.setupModal();
                this.loadExistingImages();
            });
        } else {
            this.setupDragAndDrop();
            this.setupModal();
            this.loadExistingImages();
        }
    }

    setupDragAndDrop() {
        const uploadArea = document.getElementById('uploadArea');
        const imageInput = document.getElementById('imageInput');
        
        if (!uploadArea || !imageInput) {
            console.error('Required DOM elements not found');
            return;
        }
        
        console.log('Setting up drag and drop...');
        
        uploadArea.addEventListener('click', () => {
            imageInput.click();
        });

        uploadArea.addEventListener('dragover', (e) => {
            e.preventDefault();
            uploadArea.classList.add('dragover');
        });

        uploadArea.addEventListener('dragleave', () => {
            uploadArea.classList.remove('dragover');
        });

        uploadArea.addEventListener('drop', (e) => {
            e.preventDefault();
            uploadArea.classList.remove('dragover');
            const files = e.dataTransfer.files;
            if (files.length > 0) {
                this.handleFileSelection(files[0]);
            }
        });

        imageInput.addEventListener('change', (e) => {
            if (e.target.files.length > 0) {
                this.handleFileSelection(e.target.files[0]);
            }
        });
    }

    setupModal() {
        const modal = document.getElementById('imageModal');
        if (!modal) {
            console.error('Modal element not found');
            return;
        }
        
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                this.closeModal();
            }
        });
    }

    handleFileSelection(file) {
        console.log('File selected:', file.name);
        
        if (!file.type.startsWith('image/')) {
            this.showMessage('Please select an image file (JPEG, PNG, GIF)', 'error');
            return;
        }

        this.createOriginalPreview(file);
        this.showSelectedFile(file);
        this.uploadImage(file);
    }

    createOriginalPreview(file) {
        const reader = new FileReader();
        reader.onload = (e) => {
            this.currentOriginalURL = e.target.result;
        };
        reader.readAsDataURL(file);
    }

    showSelectedFile(file) {
        const selectedFileDiv = document.getElementById('selectedFile');
        if (!selectedFileDiv) return;
        
        selectedFileDiv.innerHTML = `
            <strong>Selected file:</strong> ${file.name} (${this.formatFileSize(file.size)})
            <div style="margin-top: 10px;">
                <button class="view-original-btn" onclick="imageProcessor.viewOriginalImage()" 
                        style="padding: 5px 10px; border: none; border-radius: 3px; background: #e8f5e8; color: #2e7d32; cursor: pointer;">
                    👁️ View Original
                </button>
            </div>
            <div class="upload-progress">
                <div class="progress-bar" id="uploadProgress"></div>
            </div>
        `;
    }

    formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    async uploadImage(file) {
        const formData = new FormData();
        formData.append('image', file);

        try {
            const progressBar = document.getElementById('uploadProgress');
            if (progressBar) progressBar.style.width = '50%';

            // Реальный API вызов к вашему Gin бэкенду
            const response = await fetch('/upload', {
                method: 'POST',
                body: formData
            });

            if (progressBar) progressBar.style.width = '100%';

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || `Upload failed: ${response.status}`);
            }

            const result = await response.json();
            
            // Сохраняем оригинальное изображение и время загрузки
            const uploadTime = new Date();
            this.originalImages.set(result.id, {
                url: this.currentOriginalURL,
                uploadTime: uploadTime
            });
            
            this.addImage(result.id, result.status, file.name, uploadTime);
            
            // Начинаем опрос статуса
            this.pollImageStatus(result.id);
            
            this.showMessage(`✅ Image uploaded successfully! ID: ${result.id}`, 'success');
            
            // Очищаем выбор
            setTimeout(() => {
                const imageInput = document.getElementById('imageInput');
                const selectedFile = document.getElementById('selectedFile');
                if (imageInput) imageInput.value = '';
                if (selectedFile) selectedFile.innerHTML = '';
                this.currentOriginalURL = null;
            }, 1000);
            
        } catch (error) {
            console.error('Upload error:', error);
            this.showMessage('❌ Upload failed: ' + error.message, 'error');
            const progressBar = document.getElementById('uploadProgress');
            if (progressBar) progressBar.style.width = '0%';
        }
    }

    viewOriginalImage() {
        if (this.currentOriginalURL) {
            this.showModal('Original Image', this.currentOriginalURL, 'This is your original uploaded image');
        }
    }

    async viewProcessedImage(imageId, format = 'resized') {
        try {
            const imageUrl = `/image/${imageId}?format=${format}`;
            this.showModal(`Processed Image (${format})`, imageUrl, `This is the processed ${format} version`);
        } catch (error) {
            console.error('Error viewing processed image:', error);
            this.showMessage('❌ Failed to load processed image', 'error');
        }
    }

    showModal(title, imageSrc, info = '') {
        const modal = document.getElementById('imageModal');
        const modalImage = document.getElementById('modalImage');
        const modalTitle = document.getElementById('modalTitle');
        const modalInfo = document.getElementById('modalInfo');

        if (!modal || !modalImage || !modalTitle || !modalInfo) {
            console.error('Modal elements not found');
            return;
        }

        modalTitle.textContent = title;
        modalImage.src = imageSrc;
        modalInfo.textContent = info;
        modal.style.display = 'flex';

        modalImage.onerror = () => {
            modalInfo.innerHTML = '❌ Failed to load image. The image may still be processing.<br>Please try again later.';
            modalImage.style.display = 'none';
        };

        modalImage.onload = () => {
            modalImage.style.display = 'block';
        };
    }

    closeModal() {
        const modal = document.getElementById('imageModal');
        const modalImage = document.getElementById('modalImage');
        if (modal) modal.style.display = 'none';
        if (modalImage) modalImage.src = '';
    }

    async pollImageStatus(imageId) {
        const maxAttempts = 30;
        let attempts = 0;

        const poll = async () => {
            attempts++;
            try {
                const status = await this.getImageStatus(imageId);
                if (status) {
                    this.updateImage(imageId, status);
                    
                    if (status.status === 'completed' || attempts >= maxAttempts) {
                        if (status.status === 'completed') {
                            this.showMessage(`✅ Image ${imageId} processing completed!`, 'success');
                        } else {
                            this.showMessage(`⚠️ Image ${imageId} processing timeout`, 'error');
                        }
                        return;
                    }
                    
                    setTimeout(poll, 2000);
                }
            } catch (error) {
                console.error('Polling error:', error);
                if (attempts < maxAttempts) {
                    setTimeout(poll, 2000);
                } else {
                    this.showMessage(`❌ Failed to get status for image ${imageId}`, 'error');
                }
            }
        };

        poll();
    }

    async getImageStatus(id) {
        const response = await fetch(`/image/${id}`);
        if (!response.ok) {
            throw new Error('Failed to get image status');
        }
        return await response.json();
    }

    addImage(id, status, filename = '', uploadTime = new Date()) {
        this.images.set(id, { 
            id, 
            status, 
            filename,
            uploadTime: uploadTime
        });
        this.renderImages();
    }

    updateImage(id, data) {
        const image = this.images.get(id);
        if (image) {
            const originalUploadTime = image.uploadTime;
            Object.assign(image, data);
            image.uploadTime = originalUploadTime;
            this.renderImages();
        }
    }

    async deleteImage(id) {
        if (!confirm('Are you sure you want to delete this image?')) {
            return;
        }

        try {
            const response = await fetch(`/image/${id}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || 'Delete failed');
            }

            this.images.delete(id);
            this.originalImages.delete(id);
            this.renderImages();
            this.showMessage('✅ Image deleted successfully', 'success');
        } catch (error) {
            console.error('Delete error:', error);
            this.showMessage('❌ Delete failed: ' + error.message, 'error');
        }
    }

    async refreshImage(id) {
        try {
            const status = await this.getImageStatus(id);
            if (status) {
                this.updateImage(id, status);
                this.showMessage('✅ Status refreshed', 'success');
            }
        } catch (error) {
            console.error('Refresh error:', error);
            this.showMessage('❌ Refresh failed: ' + error.message, 'error');
        }
    }

    formatTime(uploadTime) {
        if (!uploadTime) return 'Unknown';
        
        const now = new Date();
        const diff = now - uploadTime;
        
        if (diff < 60000) {
            const seconds = Math.floor(diff / 1000);
            return `${seconds} second${seconds !== 1 ? 's' : ''} ago`;
        }
        
        if (diff < 3600000) {
            const minutes = Math.floor(diff / 60000);
            return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`;
        }
        
        if (uploadTime.toDateString() === now.toDateString()) {
            return `Today at ${uploadTime.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}`;
        }
        
        return uploadTime.toLocaleString();
    }

    renderImages() {
        const grid = document.getElementById('imagesGrid');
        if (!grid) {
            console.error('Images grid not found');
            return;
        }
        
        if (this.images.size === 0) {
            grid.innerHTML = `
                <div class="empty-state">
                    <div class="icon">🖼️</div>
                    <h3>No images uploaded yet</h3>
                    <p>Upload an image to see it here</p>
                </div>
            `;
            return;
        }

        grid.innerHTML = '';

        this.images.forEach(image => {
            const card = this.createImageCard(image);
            grid.appendChild(card);
        });
    }

    createImageCard(image) {
        const card = document.createElement('div');
        card.className = 'image-card';
        card.id = `image-${image.id}`;

        const preview = document.createElement('div');
        preview.className = 'image-preview-container';

        if (image.status === 'completed' && image.formats) {
            if (image.formats.thumbnail) {
                const img = document.createElement('img');
                img.src = `/image/${image.id}?format=thumbnail`;
                img.className = 'image-preview';
                img.alt = `Processed image ${image.id}`;
                img.onerror = () => {
                    this.showThumbnailPlaceholder(preview, image);
                };
                img.onclick = () => this.viewProcessedImage(image.id, 'thumbnail');
                preview.style.cursor = 'pointer';
                preview.appendChild(img);
            } else {
                this.showThumbnailPlaceholder(preview, image);
            }
        } else {
            const originalData = this.originalImages.get(image.id);
            if (originalData && originalData.url) {
                const img = document.createElement('img');
                img.src = originalData.url;
                img.className = 'original-preview';
                img.alt = `Original image ${image.id}`;
                img.onclick = () => this.showModal('Original Image (Processing)', originalData.url, 'Image is currently being processed');
                preview.style.cursor = 'pointer';
                preview.appendChild(img);
            } else {
                preview.innerHTML = `
                    <div class="processing-placeholder">
                        <div class="loading-spinner"></div>
                        <div>Processing...</div>
                        <small>This may take a few seconds</small>
                    </div>
                `;
            }
        }

        const info = document.createElement('div');
        info.className = 'image-info';
        info.innerHTML = `
            <div><strong>ID:</strong> ${image.id}</div>
            <div><strong>Filename:</strong> ${image.filename || 'Unknown'}</div>
            <div><strong>Status:</strong> <span class="status-${image.status}">${image.status.toUpperCase()}</span></div>
            <div><strong>Uploaded:</strong> ${this.formatTime(image.uploadTime)}</div>
        `;

        const formats = document.createElement('div');
        formats.className = 'formats-container';

        if (image.status === 'completed' && image.formats) {
            formats.innerHTML = '<div style="margin-bottom: 8px;"><strong>Download formats:</strong></div>';
            Object.keys(image.formats).forEach(format => {
                const link = document.createElement('a');
                link.className = 'format-link';
                link.href = `/image/${image.id}?format=${format}`;
                link.target = '_blank';
                link.textContent = format;
                link.download = `${image.id}_${format}.jpg`;
                formats.appendChild(link);
            });
        } else if (image.status === 'processing') {
            formats.innerHTML = '<div style="color: #6c757d; font-style: italic;">Processing formats: resize, thumbnail, watermark...</div>';
        }

        const actions = document.createElement('div');
        actions.className = 'image-actions';
        
        const originalData = this.originalImages.get(image.id);
        if (originalData && originalData.url) {
            const viewOriginalBtn = document.createElement('button');
            viewOriginalBtn.className = 'view-original-btn';
            viewOriginalBtn.innerHTML = '👁️ Original';
            viewOriginalBtn.onclick = () => this.showModal('Original Image', originalData.url, 'Your original uploaded image');
            actions.appendChild(viewOriginalBtn);
        }

        if (image.status === 'completed') {
            const viewResultBtn = document.createElement('button');
            viewResultBtn.className = 'view-result-btn';
            viewResultBtn.innerHTML = '📷 Result';
            viewResultBtn.onclick = () => this.viewProcessedImage(image.id, 'resized');
            actions.appendChild(viewResultBtn);
        }

        const refreshBtn = document.createElement('button');
        refreshBtn.className = 'refresh-btn';
        refreshBtn.innerHTML = '🔄 Refresh';
        refreshBtn.onclick = () => this.refreshImage(image.id);
        actions.appendChild(refreshBtn);

        const deleteBtn = document.createElement('button');
        deleteBtn.className = 'delete-btn';
        deleteBtn.innerHTML = '🗑️ Delete';
        deleteBtn.onclick = () => this.deleteImage(image.id);
        actions.appendChild(deleteBtn);

        card.appendChild(preview);
        card.appendChild(info);
        card.appendChild(formats);
        card.appendChild(actions);

        return card;
    }

    showThumbnailPlaceholder(preview, image) {
        preview.innerHTML = `
            <div class="processing-placeholder">
                <div style="font-size: 3rem;">🖼️</div>
                <div>Preview Available</div>
                <small>Click "View Result" to see processed image</small>
            </div>
        `;
    }

    showMessage(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        document.body.appendChild(toast);

        setTimeout(() => {
            toast.style.animation = 'slideOut 0.3s ease';
            setTimeout(() => {
                if (document.body.contains(toast)) {
                    document.body.removeChild(toast);
                }
            }, 300);
        }, 4000);
    }

    loadExistingImages() {
        console.log('Loading existing images...');
    }
}

// Глобальные функции
function uploadImage() {
    const imageInput = document.getElementById('imageInput');
    if (imageInput) {
        imageInput.click();
    }
}

function closeModal() {
    if (window.imageProcessor) {
        window.imageProcessor.closeModal();
    }
}

// Сделать функции доступными глобально
window.uploadImage = uploadImage;
window.closeModal = closeModal;