let currentParent = '';
let currentPage = 1;
let currentSort = 'created_at_desc';
const pageSize = 10;
let totalComments = 0;
let eventsBound = false;

console.log('🔧 JavaScript loaded successfully!');

// Глобальные функции
window.createComment = createComment;
window.loadComments = loadComments;
window.showReplyForm = showReplyForm;
window.hideReplyForm = hideReplyForm;
window.createReply = createReply;
window.deleteComment = deleteComment;
window.searchComments = searchComments;
window.showAllComments = showAllComments;
window.changePage = changePage;
window.changeSort = changeSort;

// Загрузка при готовности DOM
document.addEventListener('DOMContentLoaded', function() {
    console.log('📄 DOM loaded, initializing...');
    initializeApp();
});

function initializeApp() {
    console.log('🚀 Initializing application...');
    bindEvents();
    loadComments();
    console.log('✅ Application initialized successfully');
}

function bindEvents() {
    if (eventsBound) {
        console.log('⚠️ Events already bound, skipping...');
        return;
    }
    
    console.log('🔗 Binding events...');
    
    const submitBtn = document.getElementById('submitCommentBtn');
    if (submitBtn) {
        submitBtn.addEventListener('click', handleSubmitClick);
    }
    
    const searchBtn = document.getElementById('searchBtn');
    if (searchBtn) {
        searchBtn.addEventListener('click', searchComments);
    }
    
    const showAllBtn = document.getElementById('showAllBtn');
    if (showAllBtn) {
        showAllBtn.addEventListener('click', showAllComments);
    }
    
    const sortSelect = document.getElementById('sortSelect');
    if (sortSelect) {
        sortSelect.addEventListener('change', function(e) {
            changeSort(e.target.value);
        });
    }
    
    eventsBound = true;
    console.log('✅ Events binding completed');
}

function handleSubmitClick(e) {
    console.log('🎯 SUBMIT BUTTON CLICKED!');
    e.preventDefault();
    e.stopPropagation();
    
    const submitBtn = document.getElementById('submitCommentBtn');
    if (submitBtn) {
        submitBtn.disabled = true;
        submitBtn.textContent = 'Отправка...';
    }
    
    createComment().finally(() => {
        if (submitBtn) {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Отправить';
        }
    });
}

// 🔄 ИСПРАВЛЕННАЯ ФУНКЦИЯ - используем /comments для пагинации плоского списка
async function loadComments(parent = '', page = 1, sortBy = currentSort) {
    console.log('📥 Loading comments...', { parent, page, sortBy });
    currentParent = parent;
    currentPage = page;
    currentSort = sortBy;
    
    try {
        // Для корневых комментариев используем пагинацию
        // Для дочерних - дерево (чтобы видеть вложенность)
        let url;
        if (parent === '') {
            url = `/comments?parent=${parent}&page=${page}&page_size=${pageSize}&sort_by=${sortBy}`;
        } else {
            // Для подкомментариев используем tree чтобы видеть вложенность
            url = `/comments/tree?parent=${parent}`;
        }
        
        console.log('Fetching:', url);
        
        const response = await fetch(url);
        console.log('Response status:', response.status);
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const data = await response.json();
        console.log('Received data:', data);
        
        if (parent === '') {
            // Для корневых комментариев - пагинация
            totalComments = data.total || data.comments.length;
            displayComments(data.comments || data.comments);
            renderPagination(totalComments, page, pageSize);
        } else {
            // Для дочерних - дерево без пагинации
            displayComments(data.comments || data.comments);
            document.getElementById('pagination').innerHTML = '';
            document.getElementById('pageInfo').innerHTML = '';
        }
        
        hideSearchResults();
        
    } catch (error) {
        console.error('Ошибка загрузки комментариев:', error);
        document.getElementById('commentsTree').innerHTML = 
            '<p style="color: red;">Ошибка загрузки комментариев. Проверьте консоль.</p>';
    }
}

// Отображение комментариев
function displayComments(comments) {
    const container = document.getElementById('commentsTree');
    const infoContainer = document.getElementById('pageInfo');
    
    if (!comments || comments.length === 0) {
        container.innerHTML = '<p>Комментариев пока нет. Будьте первым!</p>';
        if (infoContainer) {
            infoContainer.innerHTML = '';
        }
        return;
    }

    container.innerHTML = '';
    
    // Информация о странице (только для корневых)
    if (infoContainer && currentParent === '') {
        const start = (currentPage - 1) * pageSize + 1;
        const end = Math.min(currentPage * pageSize, totalComments);
        infoContainer.innerHTML = `
            <div class="page-info">
                Показано ${start}-${end} из ${totalComments} комментариев
            </div>
        `;
    } else if (infoContainer) {
        infoContainer.innerHTML = '';
    }
    
    comments.forEach(comment => {
        renderComment(comment, 0, container);
    });
}

// Отрисовка пагинации (только для корневых)
function renderPagination(total, currentPage, pageSize) {
    const paginationContainer = document.getElementById('pagination');
    if (!paginationContainer || currentParent !== '') return;
    
    const totalPages = Math.ceil(total / pageSize);
    
    if (totalPages <= 1) {
        paginationContainer.innerHTML = '';
        return;
    }

    let html = '';
    
    // Кнопка "Назад"
    if (currentPage > 1) {
        html += `<button class="btn-pagination" onclick="changePage(${currentPage - 1})">‹ Назад</button>`;
    }
    
    // Номера страниц
    const startPage = Math.max(1, currentPage - 2);
    const endPage = Math.min(totalPages, currentPage + 2);
    
    for (let i = startPage; i <= endPage; i++) {
        if (i === currentPage) {
            html += `<span class="pagination-current">${i}</span>`;
        } else {
            html += `<button class="btn-pagination" onclick="changePage(${i})">${i}</button>`;
        }
    }
    
    // Кнопка "Вперед"
    if (currentPage < totalPages) {
        html += `<button class="btn-pagination" onclick="changePage(${currentPage + 1})">Вперед ›</button>`;
    }
    
    paginationContainer.innerHTML = html;
}

// Смена страницы
function changePage(page) {
    console.log('Changing to page:', page);
    loadComments(currentParent, page, currentSort);
}

// Сортировка
function changeSort(sortBy) {
    console.log('Changing sort to:', sortBy);
    loadComments(currentParent, 1, sortBy);
}

// Рендер комментария
function renderComment(comment, depth, container) {
    const commentDiv = document.createElement('div');
    commentDiv.className = 'comment' + (depth > 0 ? ' reply' : '');
    commentDiv.style.marginLeft = (depth * 40) + 'px';

    commentDiv.innerHTML = `
        <div class="comment-header">
            <span class="author">${escapeHtml(comment.author)}</span>
            <span class="date">${new Date(comment.created_at).toLocaleString()}</span>
        </div>
        <div class="comment-text">${escapeHtml(comment.text)}</div>
        <div class="actions">
            <button class="btn btn-reply" onclick="showReplyForm('${comment.id}')">Ответить</button>
            <button class="btn btn-delete" onclick="deleteComment('${comment.id}')">Удалить</button>
            ${currentParent === '' ? `<button class="btn btn-view" onclick="viewReplies('${comment.id}')">Показать ответы</button>` : ''}
        </div>
        <div id="reply-form-${comment.id}" class="reply-form hidden">
            <input type="text" id="reply-author-${comment.id}" placeholder="Ваше имя">
            <textarea id="reply-text-${comment.id}" placeholder="Текст ответа" rows="2"></textarea>
            <button class="btn btn-submit" onclick="createReply('${comment.id}')">Отправить ответ</button>
            <button class="btn" onclick="hideReplyForm('${comment.id}')">Отмена</button>
        </div>
    `;

    container.appendChild(commentDiv);

    // Рекурсивно отображаем дочерние комментарии (если они есть в данных)
    if (comment.children && comment.children.length > 0) {
        comment.children.forEach(child => renderComment(child, depth + 1, container));
    }
}

// Просмотр ответов
function viewReplies(commentId) {
    console.log('Viewing replies for:', commentId);
    loadComments(commentId, 1, currentSort);
}

// Создание комментария
async function createComment() {
    console.log('=== CREATE COMMENT STARTED ===');
    
    const authorInput = document.getElementById('author');
    const textInput = document.getElementById('text');
    
    const author = authorInput ? authorInput.value : '';
    const text = textInput ? textInput.value : '';

    console.log('Form values:', { author, text, currentParent });

    if (!author.trim()) {
        alert('Пожалуйста, введите ваше имя');
        authorInput?.focus();
        return;
    }
    
    if (!text.trim()) {
        alert('Пожалуйста, введите текст комментария');
        textInput?.focus();
        return;
    }

    try {
        const requestBody = {
            author: author.trim(),
            text: text.trim(),
            parent_id: currentParent || ""
        };
        
        console.log('Request body to send:', requestBody);

        const response = await fetch('/comments', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(requestBody)
        });

        console.log('Response received. Status:', response.status);

        if (response.ok) {
            const createdComment = await response.json();
            console.log('✅ Comment created successfully:', createdComment);
            
            if (authorInput) authorInput.value = '';
            if (textInput) textInput.value = '';
            
            // Перезагружаем комментарии
            await loadComments(currentParent, currentPage, currentSort);
            
        } else {
            let errorText = 'Unknown error';
            try {
                const errorData = await response.json();
                errorText = errorData.error || 'Unknown server error';
            } catch {
                errorText = await response.text();
            }
            alert('Ошибка при создании комментария: ' + errorText);
        }
    } catch (error) {
        console.error('❌ Network error:', error);
        alert('Ошибка сети: ' + error.message);
    }
}

// Показать форму ответа
function showReplyForm(commentId) {
    console.log('Showing reply form for:', commentId);
    
    document.querySelectorAll('.reply-form').forEach(form => {
        form.classList.add('hidden');
    });
    
    const replyForm = document.getElementById('reply-form-' + commentId);
    if (replyForm) {
        replyForm.classList.remove('hidden');
        
        const authorInput = document.getElementById('reply-author-' + commentId);
        if (authorInput) {
            authorInput.focus();
        }
    }
}

// Скрыть форму ответа
function hideReplyForm(commentId) {
    const replyForm = document.getElementById('reply-form-' + commentId);
    if (replyForm) {
        replyForm.classList.add('hidden');
    }
}

// Создание ответа
async function createReply(parentId) {
    console.log('Creating reply for:', parentId);
    const author = document.getElementById('reply-author-' + parentId).value;
    const text = document.getElementById('reply-text-' + parentId).value;

    if (!author || !text) {
        alert('Заполните все поля');
        return;
    }

    try {
        const response = await fetch('/comments', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                author: author,
                text: text,
                parent_id: parentId
            })
        });

        if (response.ok) {
            hideReplyForm(parentId);
            document.getElementById('reply-author-' + parentId).value = '';
            document.getElementById('reply-text-' + parentId).value = '';
            loadComments(currentParent, currentPage, currentSort);
        } else {
            const error = await response.text();
            alert('Ошибка при создании ответа: ' + error);
        }
    } catch (error) {
        console.error('Ошибка:', error);
        alert('Ошибка при создании ответа: ' + error.message);
    }
}

// Удаление комментария
async function deleteComment(commentId) {
    if (!confirm('Удалить комментарий и все ответы?')) {
        return;
    }

    try {
        const response = await fetch('/comments/' + commentId, {
            method: 'DELETE'
        });

        if (response.ok) {
            loadComments(currentParent, currentPage, currentSort);
        } else {
            const error = await response.text();
            alert('Ошибка при удалении комментария: ' + error);
        }
    } catch (error) {
        console.error('Ошибка:', error);
        alert('Ошибка при удалении комментария: ' + error.message);
    }
}

// Поиск комментариев
async function searchComments() {
    const query = document.getElementById('searchQuery').value;
    if (!query) {
        alert('Введите текст для поиска');
        return;
    }

    try {
        const response = await fetch('/comments/search?q=' + encodeURIComponent(query) + '&page=1&page_size=' + pageSize);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        displaySearchResults(data.comments, data.total, 1, query);
    } catch (error) {
        console.error('Ошибка поиска:', error);
        alert('Ошибка поиска: ' + error.message);
    }
}

// Отображение результатов поиска
function displaySearchResults(comments, total, page, query) {
    const container = document.getElementById('searchResults');
    const treeContainer = document.getElementById('commentsTree');
    
    treeContainer.classList.add('hidden');
    container.classList.remove('hidden');
    container.innerHTML = '<h3>Результаты поиска: "' + escapeHtml(query) + '"</h3>';

    if (!comments || comments.length === 0) {
        container.innerHTML += '<p>Комментарии не найдены</p>';
        return;
    }

    comments.forEach(comment => {
        const commentDiv = document.createElement('div');
        commentDiv.className = 'comment';
        commentDiv.innerHTML = `
            <div class="comment-header">
                <span class="author">${escapeHtml(comment.author)}</span>
                <span class="date">${new Date(comment.created_at).toLocaleString()}</span>
            </div>
            <div class="comment-text">${escapeHtml(comment.text)}</div>
            <div class="actions">
                <button class="btn" onclick="viewReplies('${comment.id}')">Показать в дереве</button>
                <button class="btn btn-delete" onclick="deleteComment('${comment.id}')">Удалить</button>
            </div>
        `;
        container.appendChild(commentDiv);
    });

    renderSearchPagination(total, page, pageSize, query);
}

// Пагинация для поиска
function renderSearchPagination(total, page, pageSize, query) {
    const paginationContainer = document.getElementById('pagination');
    if (!paginationContainer) return;
    
    const totalPages = Math.ceil(total / pageSize);
    
    if (totalPages <= 1) {
        paginationContainer.innerHTML = '';
        return;
    }

    let html = '';
    
    if (page > 1) {
        html += `<button class="btn-pagination" onclick="searchPage(${page - 1}, '${query}')">‹ Назад</button>`;
    }
    
    const startPage = Math.max(1, page - 2);
    const endPage = Math.min(totalPages, page + 2);
    
    for (let i = startPage; i <= endPage; i++) {
        if (i === page) {
            html += `<span class="pagination-current">${i}</span>`;
        } else {
            html += `<button class="btn-pagination" onclick="searchPage(${i}, '${query}')">${i}</button>`;
        }
    }
    
    if (page < totalPages) {
        html += `<button class="btn-pagination" onclick="searchPage(${page + 1}, '${query}')">Вперед ›</button>`;
    }
    
    paginationContainer.innerHTML = html;
}

// Смена страницы в поиске
async function searchPage(page, query) {
    try {
        const response = await fetch('/comments/search?q=' + encodeURIComponent(query) + '&page=' + page + '&page_size=' + pageSize);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        displaySearchResults(data.comments, data.total, page, query);
    } catch (error) {
        console.error('Ошибка пагинации поиска:', error);
        alert('Ошибка пагинации поиска: ' + error.message);
    }
}

// Скрыть результаты поиска
function hideSearchResults() {
    const container = document.getElementById('searchResults');
    const treeContainer = document.getElementById('commentsTree');
    
    if (container) container.classList.add('hidden');
    if (treeContainer) treeContainer.classList.remove('hidden');
}

// Показать все комментарии
function showAllComments() {
    hideSearchResults();
    loadComments('', 1, currentSort);
}

// Экранирование HTML
function escapeHtml(unsafe) {
    if (!unsafe) return '';
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

// Глобальные функции
window.searchPage = searchPage;
window.viewReplies = viewReplies;