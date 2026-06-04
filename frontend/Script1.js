// ========== ГЛОБАЛЬНЫЕ ПЕРЕМЕННЫЕ ==========
let currentUser = null;
let mockTemplates = [];
let mockRequests = [];
let executorTasks = [];
let currentDetailTask = null;
let adminUsers = [];
let adminContours = [];
let requestLogs = [];
let taskLogs = [];
let isSubmitting = false;
let activeContourFilter = null;
let activeDeadlineFilter = 'all';
let activeTaskStatusFilter = 'all';
let availableExecutors = [];
let chartsInitialized = false;
let requestsChartInstance = null;
let tasksChartInstance = null;
// Для фильтров у заказчика
let customerStatusFilter = 'all';
let customerExecutorFilter = 'all';
let customerContourFilter = 'all';
let customerSortOrder = 'asc';

const API_BASE = 'http://localhost:8080/api';
const MAX_HOURS = 200;

const statusMapping = {
    'pending': 'В планах',
    'in_progress': 'В работе',
    'completed': 'Завершено'
};

const reverseStatusMapping = {
    'В планах': 'pending',
    'В работе': 'in_progress',
    'Завершено': 'completed'
};

const requestStatusMapping = {
    'draft': 'Черновик',
    'submitted': 'Отправлена',
    'in_progress': 'В работе',
    'completed': 'Завершено',
    'overdue': 'Просрочена'
};

const deadlineFilterMap = {
    'today': { label: 'Сегодня', filter: (date) => {
        const today = new Date();
        return date.toDateString() === today.toDateString();
    }},
    'tomorrow': { label: 'Завтра', filter: (date) => {
        const tomorrow = new Date();
        tomorrow.setDate(tomorrow.getDate() + 1);
        return date.toDateString() === tomorrow.toDateString();
    }},
    'this_week': { label: 'На этой неделе', filter: (date) => {
        const now = new Date();
        const startOfWeek = new Date(now);
        startOfWeek.setDate(now.getDate() - now.getDay());
        const endOfWeek = new Date(startOfWeek);
        endOfWeek.setDate(startOfWeek.getDate() + 6);
        return date >= startOfWeek && date <= endOfWeek;
    }},
    'later': { label: 'Позже', filter: (date) => {
        const now = new Date();
        const endOfWeek = new Date(now);
        endOfWeek.setDate(now.getDate() + (7 - now.getDay()));
        return date > endOfWeek;
    }},
    'overdue': { label: 'Просрочено', filter: (date) => {
        const today = new Date();
        today.setHours(0, 0, 0, 0);
        return date < today;
    }}
};

// ========== API HELPER ==========
async function apiRequest(endpoint, method, body = null, needAuth = true) {
    const headers = { 'Content-Type': 'application/json' };
    if (needAuth) {
        const token = localStorage.getItem('token');
        if (!token) throw new Error('Нет токена');
        headers['Authorization'] = `Bearer ${token}`;
    }
    const options = { method, headers };
    if (body) options.body = JSON.stringify(body);
    const response = await fetch(`${API_BASE}${endpoint}`, options);
    if (!response.ok) {
        const error = await response.json().catch(() => ({ error: 'Ошибка сервера' }));
        throw new Error(error.error || `HTTP ${response.status}`);
    }
    if (response.status === 204) return null;
    return await response.json();
}

function saveToken(token) { localStorage.setItem('token', token); }
function clearToken() { localStorage.removeItem('token'); }

// ========== ЗАГРУЗКА ДАННЫХ ==========
async function loadTemplatesFromServer() {
    try {
        const endpoint = currentUser?.role === 'admin' ? '/admin/works' : '/works';
        const works = await apiRequest(endpoint, 'GET');
        mockTemplates = works.map(w => ({
            id: w.id,
            name: w.name,
            description: w.description || '',
            hours: w.normative_hours || 1,
        }));
        renderTemplatesList();
        renderNewRequestForm();
    } catch (error) {
        console.error('Ошибка загрузки работ:', error);
        mockTemplates = [];
    }
}

async function loadContours() {
    try {
        const contours = await apiRequest('/contours', 'GET');
        const select = document.getElementById('contourSelect');
        if (select && contours && contours.length > 0) {
            select.innerHTML = contours.map(c => `<option value="${c.id}">${escapeHtml(c.name)}${c.description ? ` — ${escapeHtml(c.description)}` : ''}</option>`).join('');
        }
        return contours;
    } catch (error) {
        console.error('Ошибка загрузки контуров:', error);
        return [];
    }
}

async function loadExecutors() {
    try {
        const executors = await apiRequest('/executors', 'GET');
        availableExecutors = executors.filter(e => e.role === 'executor').map(e => ({
            id: e.id,
            name: e.name || e.full_name || `${e.last_name || ''} ${e.first_name || ''} ${e.patronymic || ''}`.trim() || e.email,
            email: e.email,
            last_name: e.last_name,
            first_name: e.first_name,
            patronymic: e.patronymic
        }));
        return availableExecutors;
    } catch (error) {
        console.error('Ошибка загрузки исполнителей:', error);
        availableExecutors = [];
        return [];
    }
}

async function loadCustomerRequests() {
    try {
        const requests = await apiRequest('/requests', 'GET');
        mockRequests = requests.map(r => ({
            id: r.id,
            title: r.title || 'Без названия',
            contour: r.contour?.name || r.contour_name || '-',
            contour_id: r.contour_id,
            totalHours: r.total_hours || r.tasks?.reduce((sum, t) => sum + (t.work?.normative_hours || 1), 0) || 0,
            status: requestStatusMapping[r.status] || r.status,
            createdBy: r.customer?.name || r.customer_name || r.customer_id,
            created_at: r.created_at, 
            deadline: r.deadline_at || r.deadline,
            tasks: (r.tasks || []).map(t => {
                let executorName = null;
                if (t.executor?.name) {
                    executorName = t.executor.name;
                } else if (t.executor_name) {
                    executorName = t.executor_name;
                } else if (t.executor?.full_name) {
                    executorName = t.executor.full_name;
                }
                
                return {
                    id: t.id,
                    name: t.work?.name || 'Без названия',
                    hours: t.work?.normative_hours || 1,
                    description: t.work?.description || '',
                    status: statusMapping[t.status] || t.status || 'В планах',
                    comment: t.customer_comment || null,
                    executor_id: t.executor_id || t.executor?.id || null,
                    executor_name: executorName
                };
            })
        }));
        renderCustomerView();
        renderFiltersAndMetrics();
    } catch (error) {
        console.error('Ошибка загрузки заявок:', error);
        mockRequests = [];
        renderCustomerView();
    }
}

async function loadExecutorTasks() {
    try {
        const tasks = await apiRequest('/tasks', 'GET');
        const myTasks = tasks.filter(t => t.executor_id === currentUser?.id);
        
        executorTasks = myTasks.map(t => ({
            id: t.id,
            name: t.work?.name || 'Без названия',
            hours: t.work?.normative_hours || 1,
            description: t.work?.description || '',
            status: statusMapping[t.status] || t.status || 'В работе',
            requestId: t.request_id,
            requestTitle: t.request?.title || 'Без названия',
            contour: t.request?.contour?.name || '-',
            contour_description: t.request?.contour?.description || '', 
            contour_id: t.request?.contour_id,
            deadline: t.deadline_at || t.deadline || t.request?.deadline_at || t.request?.deadline,
            comment: t.customer_comment || null
        }));
        
        renderExecutorView();
        return executorTasks.filter(t => t.status !== 'Завершено');
    } catch (error) {
        console.error('Ошибка загрузки задач:', error);
        renderExecutorView();
        return [];
    }
}

// ========== УПРАВЛЕНИЕ ПОЛЬЗОВАТЕЛЯМИ (АДМИН) ==========
async function loadAdminUsers() {
    try {
        const users = await apiRequest('/admin/users', 'GET');
        adminUsers = users.filter(u => u.id !== currentUser?.id);
        renderAdminUsersList();
    } catch (error) {
        console.error('Ошибка загрузки пользователей:', error);
        adminUsers = [];
        renderAdminUsersList();
    }
}

function renderAdminUsersList() {
    const container = document.getElementById('adminUsersList');
    if (!container) return;
    
    if (adminUsers.length === 0) {
        container.innerHTML = '<div class="empty-state">Нет пользователей</div>';
        return;
    }
    
    container.innerHTML = adminUsers.map(user => {
        const isAdmin = user.role === 'admin';
        const fullName = [user.last_name, user.first_name, user.patronymic].filter(Boolean).join(' ') || user.name || user.email;
        return `
        <div class="user-card">
            <div class="user-info-text">
                <strong>${escapeHtml(fullName)}</strong><br>
                <small>Email: ${user.email} | Роль: ${user.role === 'customer' ? 'Заказчик' : user.role === 'executor' ? 'Исполнитель' : 'Админ'}</small>
            </div>
            <div class="user-actions">
                ${!isAdmin ? `
                    <button onclick="showEditUserModal(${user.id}, '${user.email}', '${escapeHtml(fullName)}', '${user.role}')" class="btn-outline">Редактировать</button>
                    <button onclick="deleteAdminUser(${user.id})" class="btn-danger">Удалить</button>
                ` : '<span style="color:#999; font-size:12px;">Системный администратор</span>'}
            </div>
        </div>
    `}).join('');
}

function showCreateUserModal() {
    document.getElementById('newUserEmail').value = '';
    document.getElementById('newUserLastName').value = '';
    document.getElementById('newUserFirstName').value = '';
    document.getElementById('newUserPatronymic').value = '';
    document.getElementById('newUserRole').value = 'customer';
    document.getElementById('newUserPassword').value = '';
    document.getElementById('createUserModal').style.display = 'flex';
}

function closeCreateUserModal() {
    document.getElementById('createUserModal').style.display = 'none';
}

async function createAdminUser() {
    const email = document.getElementById('newUserEmail').value.trim();
    const lastName = document.getElementById('newUserLastName').value.trim();
    const firstName = document.getElementById('newUserFirstName').value.trim();
    const patronymic = document.getElementById('newUserPatronymic').value.trim();
    const role = document.getElementById('newUserRole').value;
    const password = document.getElementById('newUserPassword').value;
    
    if (!email || !lastName || !firstName || !password) {
        alert('Заполните все обязательные поля');
        return;
    }
    if (!email.includes('@')) {
        alert('Введите корректный email');
        return;
    }
    if (password.length < 6) {
        alert('Пароль должен быть не менее 6 символов');
        return;
    }
    if (role === 'admin') {
        alert('Нельзя создать администратора через эту форму');
        return;
    }
    
    try {
        await apiRequest('/admin/users', 'POST', { 
            email, 
            last_name: lastName,
            first_name: firstName,
            patronymic: patronymic || null,
            role, 
            password 
        });
        closeCreateUserModal();
        await loadAdminUsers();
        alert('Пользователь создан');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

function showEditUserModal(id, email, fullname, role) {
    document.getElementById('editUserId').value = id;
    document.getElementById('editUserEmail').value = email;
    document.getElementById('editUserRole').value = role;
    document.getElementById('editUserPassword').value = '';
    
    const nameParts = fullname.trim().split(' ');
    const lastName = nameParts[0] || '';
    const firstName = nameParts[1] || '';
    const patronymic = nameParts[2] || '';
    
    document.getElementById('editUserLastName').value = lastName;
    document.getElementById('editUserFirstName').value = firstName;
    document.getElementById('editUserPatronymic').value = patronymic;
    
    document.getElementById('editUserModal').style.display = 'flex';
}

function closeEditUserModal() {
    document.getElementById('editUserModal').style.display = 'none';
}

async function saveEditUser() {
    const id = document.getElementById('editUserId').value;
    const email = document.getElementById('editUserEmail').value.trim();
    const lastName = document.getElementById('editUserLastName').value.trim();
    const firstName = document.getElementById('editUserFirstName').value.trim();
    const patronymic = document.getElementById('editUserPatronymic').value.trim();
    const role = document.getElementById('editUserRole').value;
    const password = document.getElementById('editUserPassword').value;
    
    if (!email || !lastName || !firstName) {
        alert('Заполните email, фамилию и имя');
        return;
    }
    if (!email.includes('@')) {
        alert('Введите корректный email');
        return;
    }
    
    const user = adminUsers.find(u => u.id == id);
    if (user?.role === 'admin') {
        alert('Нельзя редактировать администратора');
        closeEditUserModal();
        return;
    }
    
    const body = { 
        email, 
        last_name: lastName,
        first_name: firstName,
        patronymic: patronymic || null,
        role 
    };
    if (password && password.length >= 6) {
        body.password = password;
    }
    
    try {
        await apiRequest(`/admin/users/${id}`, 'PUT', body);
        closeEditUserModal();
        await loadAdminUsers();
        alert('Пользователь обновлён');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

async function deleteAdminUser(userId) {
    const user = adminUsers.find(u => u.id === userId);
    if (user?.role === 'admin') {
        alert('Нельзя удалить администратора');
        return;
    }
    if (userId == currentUser?.id) {
        alert('Нельзя удалить самого себя');
        return;
    }
    
    if (!confirm('Удалить пользователя? Это действие необратимо.')) return;
    try {
        await apiRequest(`/admin/users/${userId}`, 'DELETE');
        await loadAdminUsers();
        alert('Пользователь удалён');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

// ========== УПРАВЛЕНИЕ КОНТУРАМИ (АДМИН) ==========
async function loadAdminContours() {
    try {
        const contours = await apiRequest('/admin/contours', 'GET');
        adminContours = contours;
        renderAdminContoursList();
    } catch (error) {
        console.error('Ошибка загрузки контуров:', error);
        adminContours = [];
        renderAdminContoursList();
    }
}

function renderAdminContoursList() {
    const container = document.getElementById('adminContoursList');
    if (!container) return;
    
    if (adminContours.length === 0) {
        container.innerHTML = '<div class="empty-state">Нет контуров</div>';
        return;
    }
    
    container.innerHTML = adminContours.map(contour => `
        <div class="contour-card">
            <div class="contour-info-text">
                <strong>${escapeHtml(contour.name)}</strong><br>
                <small>${escapeHtml(contour.description || '—')}</small><br>
                <small>ID: ${contour.id}</small>
            </div>
            <div class="contour-actions">
                <button onclick="editAdminContour(${contour.id}, '${escapeHtml(contour.name).replace(/'/g, "\\'")}', '${escapeHtml(contour.description || '').replace(/'/g, "\\'")}')" class="btn-outline">Редактировать</button>
                <button onclick="deleteAdminContour(${contour.id})" class="btn-danger">Удалить</button>
            </div>
        </div>
    `).join('');
}

function showCreateContourModal() {
    document.getElementById('newContourName').value = '';
    document.getElementById('newContourDescription').value = '';
    document.getElementById('createContourModal').style.display = 'flex';
}

function closeCreateContourModal() {
    document.getElementById('createContourModal').style.display = 'none';
}

async function createAdminContour() {
    const name = document.getElementById('newContourName').value.trim();
    const description = document.getElementById('newContourDescription').value.trim();
    if (!name) {
        alert('Введите название контура');
        return;
    }
    try {
        await apiRequest('/admin/contours', 'POST', { name, description });
        closeCreateContourModal();
        await loadAdminContours();
        alert('Контур создан');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

function editAdminContour(id, oldName, oldDescription) {
    document.getElementById('editContourId').value = id;
    document.getElementById('editContourName').value = oldName;
    document.getElementById('editContourDescription').value = oldDescription || '';
    document.getElementById('editContourModal').style.display = 'flex';
}

function closeEditContourModal() {
    document.getElementById('editContourModal').style.display = 'none';
}

async function saveEditContour() {
    const id = document.getElementById('editContourId').value;
    const name = document.getElementById('editContourName').value.trim();
    const description = document.getElementById('editContourDescription').value.trim();
    
    if (!name) {
        alert('Введите название контура');
        return;
    }
    
    try {
        await apiRequest(`/admin/contours/${id}`, 'PUT', { name, description });
        closeEditContourModal();
        await loadAdminContours();
        alert('Контур обновлён');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

async function deleteAdminContour(id) {
    if (!id || isNaN(id)) {
        alert('Неверный ID контура');
        return;
    }
    
    if (!confirm(`Удалить контур #${id}? Это действие необратимо.`)) return;
    
    try {
        await apiRequest(`/admin/contours/${id}`, 'DELETE');
        await loadAdminContours();
        alert('Контур удалён');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

// ========== ЖУРНАЛЫ ==========
async function loadRequestLogs(limit = 10, requestId = null) {
    try {
        let url = `/admin/request-logs?limit=${limit}`;
        if (requestId) url += `&request_id=${requestId}`;
        const logs = await apiRequest(url, 'GET');
        requestLogs = logs;
        renderRequestLogs();
    } catch (error) {
        console.error('Ошибка загрузки журнала заявок:', error);
        requestLogs = [];
        renderRequestLogs();
    }
}

async function loadTaskLogs(limit = 10, taskId = null, requestId = null) {
    try {
        let url = `/admin/task-logs?limit=${limit}`;
        if (taskId) url += `&task_id=${taskId}`;
        if (requestId) url += `&request_id=${requestId}`;
        const logs = await apiRequest(url, 'GET');
        taskLogs = logs;
        renderTaskLogs();
    } catch (error) {
        console.error('Ошибка загрузки журнала задач:', error);
        taskLogs = [];
        renderTaskLogs();
    }
}

function renderRequestLogs() {
    const container = document.getElementById('requestLogsList');
    if (!container) return;
    
    if (requestLogs.length === 0) {
        container.innerHTML = '<div class="empty-state">Нет записей</div>';
        return;
    }
    
    container.innerHTML = requestLogs.map(log => `
        <div class="log-item">
            <div class="log-header">
                <strong>Заявка ${log.request_id || '-'}</strong>
                <span class="log-time">${new Date(log.created_at).toLocaleString()}</span>
            </div>
            <div class="log-content">
                <span class="log-user">Пользователь: ${log.user_email || log.user_id}</span>
                <span class="log-action">Действие: ${log.action}</span>
            </div>
            ${log.details ? `<div class="log-details">${log.details}</div>` : ''}
        </div>
    `).join('');
}

function renderTaskLogs() {
    const container = document.getElementById('taskLogsList');
    if (!container) return;
    
    if (taskLogs.length === 0) {
        container.innerHTML = '<div class="empty-state">Нет записей</div>';
        return;
    }
    
    container.innerHTML = taskLogs.map(log => `
        <div class="log-item">
            <div class="log-header">
                <strong>Задача ${log.task_id || '-'}</strong>
                <span class="log-time">${new Date(log.created_at).toLocaleString()}</span>
            </div>
            <div class="log-content">
                <span class="log-user">Пользователь: ${log.user_email || log.user_id}</span>
                <span class="log-action">${log.old_status} → ${log.new_status}</span>
            </div>
            ${log.details ? `<div class="log-details">${log.details}</div>` : ''}
        </div>
    `).join('');
}

function filterRequestLogs() {
    const requestId = document.getElementById('filterRequestId').value.trim();
    const limit = document.getElementById('filterRequestLimit').value || 50;
    loadRequestLogs(limit, requestId || null);
}

function filterTaskLogs() {
    const taskId = document.getElementById('filterTaskId').value.trim();
    const requestId = document.getElementById('filterTaskRequestId').value.trim();
    const limit = document.getElementById('filterTaskLimit').value || 50;
    loadTaskLogs(limit, taskId || null, requestId || null);
}

// ========== РЕДАКТИРОВАНИЕ РАБОТ (АДМИН) ==========
function editTemplateWork(id, name, description, hours) {
    document.getElementById('editWorkId').value = id;
    document.getElementById('editWorkName').value = name;
    document.getElementById('editWorkDesc').value = description || '';
    document.getElementById('editWorkHours').value = hours;
    document.getElementById('editWorkModal').style.display = 'flex';
}

function closeEditWorkModal() {
    document.getElementById('editWorkModal').style.display = 'none';
}

async function saveEditWork() {
    const id = document.getElementById('editWorkId').value;
    const name = document.getElementById('editWorkName').value.trim();
    const description = document.getElementById('editWorkDesc').value.trim();
    let hours = parseFloat(document.getElementById('editWorkHours').value);
    
    if (!name) { alert('Введите название работы'); return; }
    if (isNaN(hours) || hours < 1) { alert('Часы должны быть не менее 1'); return; }
    if (hours > MAX_HOURS) { alert(`Часы не могут превышать ${MAX_HOURS}`); return; }
    
    try {
        await apiRequest(`/admin/works/${id}`, 'PUT', { name, description, normative_hours: hours });
        closeEditWorkModal();
        await loadTemplatesFromServer();
        alert('Работа обновлена');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

// ========== ОТЧЁТЫ ==========
async function generateReport(requestId, format = 'json') {
    try {
        const request = mockRequests.find(r => r.id === requestId);
        if (!request) throw new Error('Заявка не найдена');
        
        const reportData = {
            request_id: request.id,
            title: request.title,
            contour: request.contour,
            status: request.status,
            created_at: request.created_at,
            deadline: request.deadline,
            total_hours: request.totalHours,
            tasks: request.tasks.map(t => ({
                name: t.name,
                description: t.description || '—',
                hours: t.hours,
                status: t.status,
                comment: t.comment ? (t.comment.length > 100 ? t.comment.substring(0, 100) + '...' : t.comment) : '—',
                executor: t.executor_name || 'Не назначен'
            }))
        };
        
        if (format === 'pdf') {
            const token = localStorage.getItem('token');
            const response = await fetch(`${API_BASE}/requests/${requestId}/report/pdf`, {
                method: 'GET',
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (!response.ok) throw new Error('Ошибка загрузки PDF');
            const blob = await response.blob();
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `report_request_${requestId}.pdf`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
            alert('PDF отчёт скачан');
        } else {
            alert(JSON.stringify(reportData, null, 2));
        }
    } catch (error) {
        alert('Ошибка формирования отчёта: ' + error.message);
    }
}

async function generateSummaryReport(format = 'json') {
    try {
        if (format === 'pdf') {
            const token = localStorage.getItem('token');
            const response = await fetch(`${API_BASE}/requests/reports/summary/pdf`, {
                method: 'GET',
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (!response.ok) throw new Error('Ошибка загрузки PDF');
            const blob = await response.blob();
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `summary_report_${Date.now()}.pdf`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
            alert('Сводный PDF-отчёт скачан');
        } else {
            const report = await apiRequest('/requests/reports/summary', 'GET');
            alert(JSON.stringify(report, null, 2));
        }
    } catch (error) {
        alert('Ошибка формирования сводного отчёта: ' + error.message);
    }
}

// ========== ПРОДЛЕНИЕ ДЕДЛАЙНА ==========
async function extendDeadline(requestId) {
    const today = new Date().toISOString().split('T')[0];
    const newDeadline = prompt(`Введите новую дату (ГГГГ-ММ-ДД):\nНе ранее ${today}`);
    if (!newDeadline) return;
    
    if (new Date(newDeadline) < new Date(today)) {
        alert('Дедлайн не может быть раньше сегодняшней даты');
        return;
    }
    
    const deadline_at = new Date(newDeadline).toISOString();
    
    try {
        await apiRequest(`/requests/${requestId}/extend-deadline`, 'POST', { deadline_at });
        await loadCustomerRequests();
        alert('Дедлайн продлён');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

// ========== НАЗНАЧЕНИЕ ИСПОЛНИТЕЛЕЙ ==========
async function assignExecutorToTask(requestId, taskId, executorId) {
    try {
        await apiRequest(`/requests/${requestId}/tasks/${taskId}/assign`, 'PUT', { executor_id: executorId });
        
        const updatedRequest = await apiRequest(`/requests/${requestId}`, 'GET');
        const requestIndex = mockRequests.findIndex(r => r.id === requestId);
        
        if (requestIndex !== -1) {
            mockRequests[requestIndex] = {
                id: updatedRequest.id,
                title: updatedRequest.title || 'Без названия',
                contour: updatedRequest.contour?.name || updatedRequest.contour_name || '-',
                contour_id: updatedRequest.contour_id,
                totalHours: updatedRequest.total_hours || updatedRequest.tasks?.reduce((sum, t) => sum + (t.work?.normative_hours || 1), 0) || 0,
                status: requestStatusMapping[updatedRequest.status] || updatedRequest.status,
                createdBy: updatedRequest.customer?.name || updatedRequest.customer_name || updatedRequest.customer_id,
                created_at: updatedRequest.created_at, 
                deadline: updatedRequest.deadline_at || updatedRequest.deadline,
                tasks: (updatedRequest.tasks || []).map(t => {
                    let executorName = null;
                    if (t.executor?.name) {
                        executorName = t.executor.name;
                    } else if (t.executor_name) {
                        executorName = t.executor_name;
                    } else if (t.executor?.full_name) {
                        executorName = t.executor.full_name;
                    }
                    
                    return {
                        id: t.id,
                        name: t.work?.name || 'Без названия',
                        hours: t.work?.normative_hours || 1,
                        status: statusMapping[t.status] || t.status || 'В планах',
                        comment: t.customer_comment || null,
                        executor_id: t.executor_id || t.executor?.id || null,
                        executor_name: executorName
                    };
                })
            };
        }
        
        const expandedState = {};
        document.querySelectorAll('.request-details').forEach(details => {
            const id = details.id.replace('request-details-', '');
            expandedState[id] = details.style.display === 'block';
        });
        
        renderCustomerView();
        
        Object.keys(expandedState).forEach(id => {
            if (expandedState[id]) {
                const detailsDiv = document.getElementById(`request-details-${id}`);
                if (detailsDiv) {
                    detailsDiv.style.display = 'block';
                    const expandIcon = document.querySelector(`.request-item[data-request-id="${id}"] .expand-icon`);
                    if (expandIcon) {
                        expandIcon.innerHTML = '▲ Свернуть';
                    }
                }
            }
        });
        
        alert('Исполнитель назначен');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

function showAssignExecutorModal(requestId, taskId, currentExecutorId) {
    const modalHtml = `
        <div id="assignExecutorModal" class="modal" style="display: flex; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); z-index: 1000; align-items: center; justify-content: center;">
            <div class="modal-content" style="max-width: 400px; width: 90%; background: white; border-radius: 12px;">
                <div style="padding: 20px; border-bottom: 1px solid #e5e7eb; display: flex; justify-content: space-between; align-items: center;">
                    <h3 style="margin: 0;">Назначить исполнителя</h3>
                    <span class="close" onclick="closeAssignExecutorModal()" style="font-size: 28px; cursor: pointer; color: #999; line-height: 1;">&times;</span>
                </div>
                <div style="padding: 20px;">
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Исполнитель</label>
                        <select id="executorSelect" style="width: 100%; padding: 8px; border-radius: 6px; border: 1px solid #d1d5db;">
                            <option value="">Не выбран</option>
                            ${availableExecutors.map(e => `<option value="${e.id}" ${currentExecutorId === e.id ? 'selected' : ''}>${escapeHtml(e.name)} (${e.email})</option>`).join('')}
                        </select>
                    </div>
                    <div style="display:flex; gap:10px; margin-top:20px;">
                        <button onclick="confirmAssignExecutor(${requestId}, ${taskId})" class="btn-primary" style="flex:1;">Назначить</button>
                        <button onclick="closeAssignExecutorModal()" class="btn-outline" style="flex:1;">Отмена</button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    const oldModal = document.getElementById('assignExecutorModal');
    if (oldModal) oldModal.remove();
    document.body.insertAdjacentHTML('beforeend', modalHtml);
}

function closeAssignExecutorModal() {
    const modal = document.getElementById('assignExecutorModal');
    if (modal) modal.remove();
}

async function confirmAssignExecutor(requestId, taskId) {
    const select = document.getElementById('executorSelect');
    const executorId = select.value ? parseInt(select.value) : null;
    closeAssignExecutorModal();
    
    if (!executorId) {
        alert('Выберите исполнителя');
        return;
    }
    
    await assignExecutorToTask(requestId, taskId, executorId);
}

// ========== МОДАЛЬНОЕ ОКНО РЕДАКТИРОВАНИЯ ЗАДАЧИ ==========
let currentEditTask = null;

function showEditTaskModal(requestId, taskId) {
    const request = mockRequests.find(r => r.id === requestId);
    if (!request) return;
    const task = request.tasks.find(t => t.id === taskId);
    if (!task) return;
    
    currentEditTask = { requestId, taskId };
    
    const modalHtml = `
        <div id="editTaskModal" class="modal" style="display: flex; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); z-index: 1000; align-items: center; justify-content: center;">
            <div class="modal-content" style="max-width: 500px; width: 90%; background: white; border-radius: 12px;">
                <div style="padding: 20px; border-bottom: 1px solid #e5e7eb; display: flex; justify-content: space-between; align-items: center;">
                    <h3 style="margin: 0;">Редактирование задачи</h3>
                    <span class="close" onclick="closeEditTaskModal()" style="font-size: 28px; cursor: pointer; color: #999; line-height: 1;">&times;</span>
                </div>
                <div style="padding: 20px;">
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Название</label>
                        <div class="value" style="background: #f3f4f6;">${escapeHtml(task.name)}</div>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Трудоёмкость (часы)</label>
                        <input type="number" id="editTaskHours" value="${task.hours}" min="1" max="${MAX_HOURS}" step="1" style="width: 100%; padding: 8px; border-radius: 6px; border: 1px solid #d1d5db;">
                        <small>От 1 до ${MAX_HOURS} часов</small>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Комментарий</label>
                        <textarea id="editTaskComment" rows="4" style="width: 100%; padding: 8px; border-radius: 6px; border: 1px solid #d1d5db; resize: vertical;">${escapeHtml(task.comment || '')}</textarea>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Исполнитель</label>
                        <select id="editTaskExecutor" style="width: 100%; padding: 8px; border-radius: 6px; border: 1px solid #d1d5db;">
                            <option value="">Не назначен</option>
                            ${availableExecutors.map(e => `<option value="${e.id}" ${task.executor_id === e.id ? 'selected' : ''}>${escapeHtml(e.name)} (${e.email})</option>`).join('')}
                        </select>
                    </div>
                    <div style="display:flex; gap:10px; margin-top:20px;">
                        <button onclick="saveEditTask()" class="btn-primary" style="flex:1;">Сохранить</button>
                        <button onclick="closeEditTaskModal()" class="btn-outline" style="flex:1;">Отмена</button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    const oldModal = document.getElementById('editTaskModal');
    if (oldModal) oldModal.remove();
    document.body.insertAdjacentHTML('beforeend', modalHtml);
}

function closeEditTaskModal() {
    const modal = document.getElementById('editTaskModal');
    if (modal) modal.remove();
    currentEditTask = null;
}

async function saveEditTask() {
    if (!currentEditTask) return;
    
    const newHours = parseInt(document.getElementById('editTaskHours').value);
    const newComment = document.getElementById('editTaskComment').value.trim();
    const newExecutorId = document.getElementById('editTaskExecutor').value;
    
    if (isNaN(newHours) || newHours < 1 || newHours > MAX_HOURS) {
        alert(`Трудоёмкость должна быть от 1 до ${MAX_HOURS} часов`);
        return;
    }
    
    const request = mockRequests.find(r => r.id === currentEditTask.requestId);
    if (!request) return;
    const task = request.tasks.find(t => t.id === currentEditTask.taskId);
    if (!task) return;
    
    task.hours = newHours;
    task.comment = newComment || null;
    
    if (newExecutorId) {
        task.executor_id = parseInt(newExecutorId);
        const executor = availableExecutors.find(e => e.id === parseInt(newExecutorId));
        task.executor_name = executor?.name || null;
        await assignExecutorToTask(currentEditTask.requestId, currentEditTask.taskId, parseInt(newExecutorId));
    }
    
    request.totalHours = request.tasks.reduce((sum, t) => sum + t.hours, 0);
    
    closeEditTaskModal();
    renderCustomerView();
    alert('Задача обновлена');
}

// ========== МОДАЛЬНОЕ ОКНО ДОБАВЛЕНИЯ КОММЕНТАРИЯ ==========
let currentCommentTask = null;

function showCommentModal(requestId, taskId) {
    currentCommentTask = { requestId, taskId };
    
    const modalHtml = `
        <div id="commentModal" class="modal" style="display: flex; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); z-index: 1000; align-items: center; justify-content: center;">
            <div class="modal-content" style="max-width: 500px; width: 90%; background: white; border-radius: 12px;">
                <div style="padding: 20px; border-bottom: 1px solid #e5e7eb; display: flex; justify-content: space-between; align-items: center;">
                    <h3 style="margin: 0;">Добавить комментарий к задаче</h3>
                    <span class="close" onclick="closeCommentModal()" style="font-size: 28px; cursor: pointer; color: #999; line-height: 1;">&times;</span>
                </div>
                <div style="padding: 20px;">
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Комментарий</label>
                        <textarea id="commentText" rows="4" style="width: 100%; padding: 8px; border-radius: 6px; border: 1px solid #d1d5db; resize: vertical;" placeholder="Введите комментарий..."></textarea>
                    </div>
                    <div style="display:flex; gap:10px; margin-top:20px;">
                        <button onclick="saveComment()" class="btn-primary" style="flex:1;">Сохранить</button>
                        <button onclick="closeCommentModal()" class="btn-outline" style="flex:1;">Отмена</button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    const oldModal = document.getElementById('commentModal');
    if (oldModal) oldModal.remove();
    document.body.insertAdjacentHTML('beforeend', modalHtml);
}

function closeCommentModal() {
    const modal = document.getElementById('commentModal');
    if (modal) modal.remove();
    currentCommentTask = null;
}

async function saveComment() {
    if (!currentCommentTask) return;
    
    const comment = document.getElementById('commentText').value.trim();
    if (!comment) {
        alert('Введите комментарий');
        return;
    }
    
    const request = mockRequests.find(r => r.id === currentCommentTask.requestId);
    if (request) {
        const task = request.tasks.find(t => t.id === currentCommentTask.taskId);
        if (task) {
            task.comment = comment;
            try {
                await apiRequest(`/requests/${currentCommentTask.requestId}/tasks/${currentCommentTask.taskId}/comment`, 'PUT', { comment });
            } catch (error) {
                console.warn('Бэкенд не поддерживает комментарии');
            }
        }
    }
    
    closeCommentModal();
    renderCustomerView();
    alert('Комментарий добавлен');
}

// ========== КОММЕНТАРИИ ==========
function truncateComment(comment, maxLength = 150) {
    if (!comment) return '';
    if (comment.length <= maxLength) return comment;
    return comment.substring(0, maxLength) + '...';
}

function getDisplayComment(task) {
    if (!task.comment) return null;
    return truncateComment(task.comment, 150);
}

// ========== РЕДАКТИРОВАНИЕ ЗАДАЧ (СТАРЫЙ МЕТОД - ЗАМЕНЁН) ==========
function editTask(requestId, taskId) {
    showEditTaskModal(requestId, taskId);
}

async function deleteTaskFromRequest(requestId, taskId) {
    if (!confirm('Удалить задачу из заявки?')) return;
    
    const request = mockRequests.find(r => r.id === requestId);
    if (request) {
        request.tasks = request.tasks.filter(t => t.id !== taskId);
        request.totalHours = request.tasks.reduce((sum, t) => sum + t.hours, 0);
        
        try {
            await apiRequest(`/requests/${requestId}/tasks/${taskId}`, 'DELETE');
        } catch (error) {
            console.warn('Ошибка удаления на бэкенде, удалено только локально');
        }
        
        renderCustomerView();
        alert('Задача удалена');
    }
}

async function addTasksToRequest(requestId, worksData) {
    let tasksToSend;
    let hasComments = false;
    
    if (Array.isArray(worksData) && worksData.length > 0) {
        if (typeof worksData[0] === 'number') {
            const uniqueIds = [...new Set(worksData)];
            tasksToSend = { work_ids: uniqueIds };
        } else if (typeof worksData[0] === 'object' && worksData[0].work_id) {
            hasComments = true;
            const uniqueMap = new Map();
            for (const item of worksData) {
                if (!uniqueMap.has(item.work_id)) {
                    uniqueMap.set(item.work_id, {
                        work_id: item.work_id,
                        comment: item.comment || null,
                        executor_id: item.executor_id || null
                    });
                }
            }
            const uniqueTasks = Array.from(uniqueMap.values());
            tasksToSend = { tasks: uniqueTasks };
        } else {
            throw new Error('Неверный формат данных');
        }
    } else {
        throw new Error('Список работ пуст');
    }
    
    try {
        await apiRequest(`/requests/${requestId}/tasks`, 'POST', tasksToSend);
        return { success: true, hasComments };
    } catch (error) {
        let errorMessage = error.message;
        if (errorMessage.includes('duplicate work_id')) {
            alert('Некоторые работы уже добавлены в план');
        } else if (errorMessage.includes('work_ids') && hasComments) {
            console.warn('Бэкенд не поддерживает комментарии при добавлении, отправляем без них');
            const fallbackIds = worksData.map(item => 
                typeof item === 'number' ? item : item.work_id
            );
            const uniqueIds = [...new Set(fallbackIds)];
            try {
                await apiRequest(`/requests/${requestId}/tasks`, 'POST', { work_ids: uniqueIds });
                return { success: true, hasComments: false, fallback: true };
            } catch (fallbackError) {
                alert('Ошибка добавления работ: ' + fallbackError.message);
                return { success: false, error: fallbackError.message };
            }
        } else {
            alert('Ошибка добавления работ: ' + errorMessage);
        }
        return { success: false, error: errorMessage };
    }
}

// ========== РЕГИСТРАЦИЯ ==========
function showRegistrationModal() {
    document.getElementById('regEmail').value = '';
    document.getElementById('regLastName').value = '';
    document.getElementById('regFirstName').value = '';
    document.getElementById('regPatronymic').value = '';
    document.getElementById('regPassword').value = '';
    document.getElementById('regConfirmPassword').value = '';
    document.getElementById('registrationModal').style.display = 'flex';
}

function closeRegistrationModal() {
    document.getElementById('registrationModal').style.display = 'none';
}

async function registerUser() {
    const email = document.getElementById('regEmail').value.trim();
    const lastName = document.getElementById('regLastName').value.trim();
    const firstName = document.getElementById('regFirstName').value.trim();
    const patronymic = document.getElementById('regPatronymic').value.trim();
    const role = document.getElementById('regRole').value;
    const password = document.getElementById('regPassword').value;
    const confirm = document.getElementById('regConfirmPassword').value;
    
    if (!email || !lastName || !firstName || !password) { 
        alert('Заполните все обязательные поля'); 
        return; 
    }
    if (password !== confirm) { 
        alert('Пароли не совпадают'); 
        return; 
    }
    if (!email.includes('@')) { 
        alert('Введите корректный email'); 
        return; 
    }
    
    try {
        const data = await apiRequest('/register', 'POST', { 
            email, 
            last_name: lastName,
            first_name: firstName,
            patronymic: patronymic || null,
            role, 
            password 
        }, false);
        
        if (data.token) {
            saveToken(data.token);
            currentUser = { 
                id: data.user.id, 
                login: data.user.email, 
                fullname: data.user.full_name || `${lastName} ${firstName} ${patronymic}`.trim(),
                role: data.user.role 
            };
            closeRegistrationModal();
            updateUIAfterLogin();
            alert(`Регистрация успешна! Добро пожаловать, ${firstName} ${lastName}`);
        }
    } catch (error) {
        alert('Ошибка регистрации: ' + error.message);
    }
}

// ========== АВТОРИЗАЦИЯ ==========
async function login(email, password) {
    if (!email.includes('@')) { alert('Введите корректный email'); return false; }
    try {
        const data = await apiRequest('/login', 'POST', { email, password }, false);
        if (data.token) {
            saveToken(data.token);
            currentUser = { id: data.user.id, login: data.user.email, fullname: data.user.name || data.user.email, role: data.user.role };
            updateUIAfterLogin();
            return true;
        }
        return false;
    } catch (error) {
        alert('Ошибка входа: ' + error.message);
        return false;
    }
}

async function updateUIAfterLogin() {
    const roleLabel = currentUser.role === 'admin' ? 'Админ' : (currentUser.role === 'customer' ? 'Заказчик' : 'Исполнитель');
    document.getElementById('userName').innerHTML = `<strong>${currentUser.fullname}</strong> <span class="role-badge">${roleLabel}</span>`;
    document.getElementById('logoutBtn').style.display = 'inline-block';
    document.getElementById('loginPanel').style.display = 'none';
    document.getElementById('adminPanel').style.display = currentUser.role === 'admin' ? 'block' : 'none';
    document.getElementById('customerPanel').style.display = currentUser.role === 'customer' ? 'block' : 'none';
    document.getElementById('executorPanel').style.display = currentUser.role === 'executor' ? 'block' : 'none';
    
    if (currentUser.role === 'admin') {
        renderAdminPanel();
        await loadTemplatesFromServer();
        await loadAdminUsers();
        await loadAdminContours();
    } else if (currentUser.role === 'customer') {
        await loadContours();
        await loadExecutors();
        await loadTemplatesFromServer();
        await loadCustomerRequests();
    } else if (currentUser.role === 'executor') {
        await loadExecutorTasks();
    }
    initTabs();
}

function logout() {
    clearToken();
    currentUser = null;
    document.getElementById('userName').innerHTML = 'Не авторизован';
    document.getElementById('logoutBtn').style.display = 'none';
    document.getElementById('loginPanel').style.display = 'block';
    document.getElementById('adminPanel').style.display = 'none';
    document.getElementById('customerPanel').style.display = 'none';
    document.getElementById('executorPanel').style.display = 'none';
}

// ========== АДМИН ==========
async function renderAdminPanel() {
    document.querySelectorAll('.admin-tab').forEach(btn => {
        btn.onclick = async () => {
            document.querySelectorAll('.admin-tab').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            const tab = btn.getAttribute('data-tab');
            document.getElementById('tabUsers').style.display = tab === 'users' ? 'block' : 'none';
            document.getElementById('tabTemplates').style.display = tab === 'templates' ? 'block' : 'none';
            document.getElementById('tabContours').style.display = tab === 'contours' ? 'block' : 'none';
            document.getElementById('tabLogs').style.display = tab === 'logs' ? 'block' : 'none';
            
            if (tab === 'logs') {
                await loadRequestLogs(50);
                await loadTaskLogs(50);
            }
        };
    });
    
    renderTemplatesList();
    await loadAdminContours();
}

function renderTemplatesList() {
    const tbody = document.getElementById('templatesTable');
    if (!tbody) return;
    tbody.innerHTML = mockTemplates.map(t => `
        <tr>
            <td><strong>${t.name}</strong><br><small>${t.description}</small></td>
            <td>${t.hours} ч</td>
            <td>
                <button onclick="editTemplateWork(${t.id}, '${t.name.replace(/'/g, "\\'")}', '${t.description.replace(/'/g, "\\'")}', ${t.hours})" class="btn-outline" style="margin-right:5px;">Редактировать</button>
                <button onclick="deleteTemplate(${t.id})" class="btn-danger">Удалить</button>
            </td>
        </tr>
    `).join('');
}

async function addTemplateWork() {
    const name = document.getElementById('newTaskName')?.value.trim();
    const desc = document.getElementById('newTaskDesc')?.value.trim();
    let hours = parseFloat(document.getElementById('newTaskHours')?.value);
    if (!name || !desc) { alert('Заполните название и описание работы'); return; }
    if (isNaN(hours) || hours < 1) { alert('Нормативные часы должны быть не менее 1'); return; }
    if (hours > MAX_HOURS) { alert(`Часы не могут превышать ${MAX_HOURS}`); return; }
    try {
        await apiRequest('/admin/works', 'POST', { name, description: desc, normative_hours: hours });
        await loadTemplatesFromServer();
        document.getElementById('newTaskName').value = '';
        document.getElementById('newTaskDesc').value = '';
        document.getElementById('newTaskHours').value = '';
        alert('Работа добавлена в справочник');
    } catch (error) {
        alert('Ошибка добавления: ' + error.message);
    }
}

async function deleteTemplate(id) {
    if (!confirm('Удалить эту работу?')) return;
    try {
        await apiRequest(`/admin/works/${id}`, 'DELETE');
        await loadTemplatesFromServer();
        alert('Работа удалена');
    } catch (error) {
        alert('Ошибка удаления: ' + error.message);
    }
}
function renderSummaryReportButton() {
    const container = document.getElementById('customerRequests');
    if (!container) return;
    
    const existingBtn = document.getElementById('summaryReportBtn');
    if (existingBtn) existingBtn.remove();
    
    const btnHtml = `
        <div id="summaryReportBtn" style="margin-bottom: 20px; display: flex; gap: 10px;">
            <button onclick="generateSummaryReport('json')" class="btn-outline">Сводный отчёт (JSON)</button>
            <button onclick="generateSummaryReport('pdf')" class="btn-outline">Сводный отчёт (PDF)</button>
        </div>
    `;
    container.insertAdjacentHTML('beforebegin', btnHtml);
}

// ========== ЗАКАЗЧИК ==========
function renderCustomerView() {
    renderActiveTasksForCustomer();  // сначала заявки
    renderNewRequestForm();
    renderSummaryReportButton();
    renderFiltersAndMetrics();  // потом метрики и фильтры
}

function renderFiltersAndMetrics() {
    const container = document.getElementById('customerRequests');
    if (!container) return;
    
    // Уникальные контуры и исполнители для фильтров
    const contours = [...new Set(mockRequests.map(r => r.contour).filter(c => c && c !== '-'))];
    const executors = [...new Set(mockRequests.flatMap(r => r.tasks.map(t => t.executor_name)).filter(e => e))];
    
    const filterHtml = `
        <div id="customerFilterSection" class="filter-section" style="margin-bottom:20px; padding:12px; background:#f9fafb; border-radius:8px; display:flex; gap:16px; flex-wrap:wrap; align-items:flex-end;">
            <div>
                <label style="font-weight:500;">Статус заявки:</label>
                <select id="customerStatusFilter" onchange="applyCustomerFilters()" style="padding:6px 12px; border-radius:6px; border:1px solid #d1d5db; margin-left:8px;">
                    <option value="all">Все</option>
                    <option value="Черновик">Черновик</option>
                    <option value="Отправлена">Отправлена</option>
                    <option value="В работе">В работе</option>
                    <option value="Просрочена">Просрочена</option>
                    <option value="Завершено">Завершено</option>
                </select>
            </div>
            <div>
                <label style="font-weight:500;">Исполнитель:</label>
                <select id="customerExecutorFilter" onchange="applyCustomerFilters()" style="padding:6px 12px; border-radius:6px; border:1px solid #d1d5db; margin-left:8px;">
                    <option value="all">Все</option>
                    ${executors.map(e => `<option value="${e}">${e}</option>`).join('')}
                </select>
            </div>
            <div>
                <label style="font-weight:500;">Контур:</label>
                <select id="customerContourFilter" onchange="applyCustomerFilters()" style="padding:6px 12px; border-radius:6px; border:1px solid #d1d5db; margin-left:8px;">
                    <option value="all">Все</option>
                    ${contours.map(c => `<option value="${c}">${c}</option>`).join('')}
                </select>
            </div>
            <div>
                <label style="font-weight:500;">Сортировка по дедлайну:</label>
                <select id="customerSortOrder" onchange="applyCustomerFilters()" style="padding:6px 12px; border-radius:6px; border:1px solid #d1d5db; margin-left:8px;">
                    <option value="none">Без сортировки</option>
                    <option value="asc">Сначала ближайшие</option>
                    <option value="desc">Сначала дальние</option>
                </select>
            </div>
        </div>
    `;
    
    // Метрики
    const totalRequests = mockRequests.length;
    const completedRequests = mockRequests.filter(r => r.status === 'Завершено').length;
    const inProgressRequests = mockRequests.filter(r => r.status === 'В работе').length;
    const draftRequests = mockRequests.filter(r => r.status === 'Черновик').length;
    const overdueRequests = mockRequests.filter(r => r.status === 'Просрочена').length;
    
    const allTasks = mockRequests.flatMap(r => r.tasks);
    const totalTasks = allTasks.length;
    const completedTasks = allTasks.filter(t => t.status === 'Завершено').length;
    const inProgressTasks = allTasks.filter(t => t.status === 'В работе').length;
    const plannedTasks = allTasks.filter(t => t.status === 'В планах').length;
    
    // Статистика по исполнителям
    const executorStats = {};
    allTasks.forEach(task => {
        if (task.executor_name) {
            if (!executorStats[task.executor_name]) {
                executorStats[task.executor_name] = { tasks: 0, hours: 0 };
            }
            executorStats[task.executor_name].tasks++;
            executorStats[task.executor_name].hours += task.hours;
        }
    });
    
    const metricsHtml = `
        <div id="metricsPanel" style="margin-bottom:20px; padding:12px; background:#fff; border-radius:8px; border:1px solid #e0e0e0;">
            <h3>📊 Метрики</h3>
            <div style="display:flex; flex-wrap:wrap; gap:20px; margin-top:10px;">
                <div style="flex:1; min-width:200px;">
                    <canvas id="requestsChart" width="200" height="200" style="max-width:200px; max-height:200px;"></canvas>
                    <p style="text-align:center">Заявки: всего ${totalRequests}</p>
                </div>
                <div style="flex:1; min-width:200px;">
                    <canvas id="tasksChart" width="200" height="200" style="max-width:200px; max-height:200px;"></canvas>
                    <p style="text-align:center">Задачи: всего ${totalTasks}</p>
                </div>
                <div style="flex:2; min-width:250px;">
                    <h4>Задачи по исполнителям</h4>
                    <table class="data-table" style="width:100%">
                        <thead>
                            <tr><th>Исполнитель</th><th>Задач</th><th>Часов</th></tr>
                        </thead>
                        <tbody>
                            ${Object.entries(executorStats).map(([name, stats]) => `
                                <tr><td style="padding:8px;">${escapeHtml(name)}</td><td style="padding:8px;">${stats.tasks}</td><td style="padding:8px;">${stats.hours}</td></tr>
                            `).join('')}
                            ${Object.keys(executorStats).length === 0 ? '<tr><td colspan="3" style="padding:8px;">Нет данных</td></tr>' : ''}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    `;
    
    // Обновляем или создаём панель метрик
    let metricsPanel = document.getElementById('metricsPanel');
    if (metricsPanel) {
        metricsPanel.remove();
    }
    container.insertAdjacentHTML('beforebegin', metricsHtml);
    
    // Обновляем или создаём фильтры
    let filterSection = document.getElementById('customerFilterSection');
    if (filterSection) {
        filterSection.remove();
    }
    container.insertAdjacentHTML('beforebegin', filterHtml);
    
    // Пересоздаём диаграммы (уничтожаем старые)
    if (requestsChartInstance) {
        requestsChartInstance.destroy();
        requestsChartInstance = null;
    }
    if (tasksChartInstance) {
        tasksChartInstance.destroy();
        tasksChartInstance = null;
    }
    
    // Создаём новые диаграммы с задержкой
    setTimeout(() => {
        if (typeof Chart !== 'undefined') {
            const requestsCtx = document.getElementById('requestsChart')?.getContext('2d');
            if (requestsCtx) {
                requestsChartInstance = new Chart(requestsCtx, {
                    type: 'doughnut',
                    data: {
                        labels: ['Завершено', 'В работе', 'Черновик', 'Просрочено'],
                        datasets: [{
                            data: [completedRequests, inProgressRequests, draftRequests, overdueRequests],
                            backgroundColor: ['#059669', '#dbeafe', '#e0e7ff', '#fee2e2'],
                            borderColor: '#fff',
                            borderWidth: 2
                        }]
                    },
                    options: { responsive: true, maintainAspectRatio: true }
                });
            }
            
            const tasksCtx = document.getElementById('tasksChart')?.getContext('2d');
            if (tasksCtx) {
                tasksChartInstance = new Chart(tasksCtx, {
                    type: 'doughnut',
                    data: {
                        labels: ['Завершено', 'В работе', 'В планах'],
                        datasets: [{
                            data: [completedTasks, inProgressTasks, plannedTasks],
                            backgroundColor: ['#059669', '#dbeafe', '#e5e7eb'],
                            borderColor: '#fff',
                            borderWidth: 2
                        }]
                    },
                    options: { responsive: true, maintainAspectRatio: true }
                });
            }
        } else {
            console.warn('Chart.js не загружен');
        }
    }, 100);

}

function applyCustomerFilters() {
    customerStatusFilter = document.getElementById('customerStatusFilter')?.value || 'all';
    customerExecutorFilter = document.getElementById('customerExecutorFilter')?.value || 'all';
    customerContourFilter = document.getElementById('customerContourFilter')?.value || 'all';
    customerSortOrder = document.getElementById('customerSortOrder')?.value || 'none';
    renderActiveTasksForCustomer();
}

function getFilteredAndSortedRequests() {
    let filtered = [...mockRequests];
    
    if (customerStatusFilter !== 'all') {
        filtered = filtered.filter(r => r.status === customerStatusFilter);
    }
    
    if (customerContourFilter !== 'all') {
        filtered = filtered.filter(r => r.contour === customerContourFilter);
    }
    
    if (customerExecutorFilter !== 'all') {
        filtered = filtered.filter(r => 
            r.tasks.some(t => t.executor_name === customerExecutorFilter)
        );
    }
    
    if (customerSortOrder === 'asc') {
        filtered.sort((a, b) => {
            if (!a.deadline) return 1;
            if (!b.deadline) return -1;
            return new Date(a.deadline) - new Date(b.deadline);
        });
    } else if (customerSortOrder === 'desc') {
        filtered.sort((a, b) => {
            if (!a.deadline) return -1;
            if (!b.deadline) return 1;
            return new Date(b.deadline) - new Date(a.deadline);
        });
    }
    
    return filtered;
}

function renderActiveTasksForCustomer() {
    const container = document.getElementById('customerRequests');
    if (!container) return;
    
    const sortedRequests = getFilteredAndSortedRequests();
    
    if (sortedRequests.length === 0) {
        container.innerHTML = '<div class="empty-state">Нет активных заявок</div>';
        return;
    }
    
    container.innerHTML = sortedRequests.map(req => {
        const tasksHtml = req.tasks.map(task => {
            const displayComment = getDisplayComment(task);
            const canEdit = (req.status === 'Черновик' || req.status === 'В планах' || req.status === 'Отправлена') && task.status === 'В планах';
            const canAddComment = (req.status === 'Черновик' || req.status === 'В планах' || req.status === 'Отправлена') && task.status === 'В планах';
            
            return `
                <div class="task-card">
                    <div class="task-header">
                        <span class="task-title">${escapeHtml(task.name || '')}</span>
                        <span class="status-badge 
                            ${task.status === 'В планах' ? 'status-planned' : 
                              task.status === 'В работе' ? 'status-progress' : 
                              task.status === 'Завершено' ? 'status-done' : ''}
                        ">${task.status}</span>
                    </div>
                    <div class="task-meta">
                        <span>Время: ${task.hours || 0} ч</span>
                        ${task.executor_name ? `<span style="margin-left:12px;">Исполнитель: ${escapeHtml(String(task.executor_name))}</span>` : '<span style="margin-left:12px; color:#e31e24;">Исполнитель не назначен</span>'}
                    </div>
                    ${displayComment ? `
                        <div class="task-comment" style="margin-top:8px; font-size:12px; color:#666; background:#fef3c7; padding:8px; border-radius:6px;">
                            Комментарий: ${escapeHtml(String(displayComment))}
                        </div>
                    ` : ''}
                    <div style="margin-top:10px;">
                        <button onclick="showTaskDetailForCustomer(${req.id}, ${task.id})" class="btn-outline" style="padding:4px 12px; font-size:12px;">Детали задачи</button>
                        ${canEdit ? `
                            <button onclick="editTask(${req.id}, ${task.id})" class="btn-outline" style="margin-left:8px; padding:4px 12px; font-size:12px;">Редактировать</button>
                            <button onclick="deleteTaskFromRequest(${req.id}, ${task.id})" class="btn-danger" style="margin-left:8px; padding:4px 12px; font-size:12px;">Удалить задачу</button>
                        ` : ''}
                        ${canAddComment && !task.comment ? `
                            <button onclick="showCommentModal(${req.id}, ${task.id})" class="btn-outline" style="margin-left:8px; padding:4px 12px; font-size:12px;">Добавить комментарий</button>
                        ` : ''}
                    </div>
                </div>
            `;
        }).join('');
        
        return `
            <div class="request-item" data-request-id="${req.id}">
                <div class="request-header" onclick="toggleRequestDetails(${req.id})" style="cursor:pointer;">
                    <div style="display:flex; justify-content:space-between; align-items:center; flex-wrap:wrap; gap:10px;">
                        <div>
                            <strong style="font-size: 16px;">${escapeHtml(req.title || '')}</strong>
                            <span class="request-title" style="font-size: 13px; color: #666;">Заявка №${req.id}</span>
                            ${req.deadline ? `<span class="deadline-badge">до ${new Date(req.deadline).toLocaleDateString()}</span>` : '<span class="deadline-badge" style="background:#e5e7eb;">дедлайн не указан</span>'}
                        </div>
                        <div>
                            <span>Контур: ${escapeHtml(req.contour || '')}</span>
                            <span class="status-badge 
                                ${req.status === 'Черновик' ? 'status-draft' : 
                                  req.status === 'В планах' ? 'status-planned' : 
                                  req.status === 'В работе' ? 'status-progress' : 
                                  req.status === 'Просрочена' ? 'status-overdue' : 
                                  req.status === 'Завершено' ? 'status-done' : ''}
                            " style="margin-left:10px;">${req.status}</span>
                        </div>
                    </div>
                    <div style="margin-top:5px; font-size:12px; color:#666;">
                        <span>Общее время: ${req.totalHours || 0} ч</span>
                    </div>
                    <div class="expand-icon" style="font-size:12px; margin-top:5px;">
                        ▼ Развернуть
                    </div>
                </div>
                <div id="request-details-${req.id}" class="request-details" style="display:none; margin-top:15px;">
                    <div class="request-info-block" style="background:#f0f0f0; padding:12px; border-radius:8px; margin-bottom:15px;">
                        <strong>Информация о заявке</strong><br>
                        <span>Создана: ${req.created_at ? new Date(req.created_at).toLocaleDateString('ru-RU') : '—'}</span><br>
                        <span>Дедлайн: ${req.deadline ? new Date(req.deadline).toLocaleDateString('ru-RU') : 'не указан'}</span><br>
                        <span>Статус: ${req.status || ''}</span><br>
                        <span>Заказчик: ${escapeHtml(req.createdBy || '')}</span>
                    </div>
                    <div class="tasks-list">
                        <h4>Задачи:</h4>
                        ${tasksHtml}
                    </div>
                    <div style="margin-top:15px;">
                        ${req.status === 'Черновик' ? `
                            <button onclick="editRequest(${req.id})" class="btn-outline" style="margin-right:10px;">Редактировать заявку</button>
                            <button onclick="deleteRequest(${req.id})" class="btn-danger" style="margin-right:10px;">Удалить заявку</button>
                            <button onclick="submitRequest(${req.id})" class="btn-primary" id="submit-btn-${req.id}">Отправить</button>
                        ` : ''}
                        ${req.status === 'Просрочена' && req.deadline ? `
                            <button onclick="extendDeadline(${req.id})" class="btn-outline" style="margin-right:10px;">Продлить дедлайн</button>
                        ` : ''}
                        <div style="display:flex; align-items:center; gap:10px; margin-top:10px;">
                            <select id="report-format-${req.id}" style="padding: 6px 12px; border-radius: 6px; border: 1px solid #d1d5db;">
                                <option value="json">JSON</option>
                                <option value="pdf">PDF</option>
                            </select>
                            <button onclick="generateReport(${req.id}, document.getElementById('report-format-${req.id}').value)" class="btn-outline">Скачать отчёт</button>
                        </div>
                    </div>
                </div>
            </div>
        `;
    }).join('');
}

function toggleRequestDetails(requestId) {
    const detailsDiv = document.getElementById(`request-details-${requestId}`);
    if (detailsDiv) {
        const isVisible = detailsDiv.style.display === 'block';
        detailsDiv.style.display = isVisible ? 'none' : 'block';
        const expandIcon = document.querySelector(`.request-item[data-request-id="${requestId}"] .expand-icon`);
        if (expandIcon) {
            expandIcon.innerHTML = isVisible ? '▼ Развернуть' : '▲ Свернуть';
        }
    }
}

function renderNewRequestForm() {
    const container = document.getElementById('templatesListNew');
    if (!container) return;
    
    if (mockTemplates.length === 0) {
        container.innerHTML = '<div class="empty-state">Нет активных типовых работ</div>';
        return;
    }
    
    const today = new Date().toISOString().split('T')[0];
    
    container.innerHTML = mockTemplates.map(t => `
        <div class="work-item" style="margin: 12px 0; padding: 10px; background: #f9fafb; border-radius: 8px;">
            <label style="display: flex; align-items: flex-start; gap: 12px;">
                <input type="checkbox" value="${t.id}" class="template-checkbox" style="margin-top: 2px;">
                <div style="flex: 1;">
                    <div><strong>${escapeHtml(t.name)}</strong><br><small>${escapeHtml(t.description)} — ${t.hours} ч</small></div>
                    <textarea 
                        class="work-comment" 
                        data-work-id="${t.id}"
                        rows="2" 
                        placeholder="Комментарий к задаче (необязательно)"
                        style="width: 100%; margin-top: 8px; padding: 6px; border-radius: 4px; border: 1px solid #d1d5db; font-size: 12px; font-family: inherit; resize: vertical;"></textarea>
                    <select class="work-executor" data-work-id="${t.id}" style="width: 100%; margin-top: 8px; padding: 6px; border-radius: 4px; border: 1px solid #d1d5db;">
                        <option value="">Не назначен</option>
                        ${availableExecutors.map(e => `<option value="${e.id}">${escapeHtml(e.name)} (${e.email})</option>`).join('')}
                    </select>
                </div>
            </label>
        </div>
    `).join('');
    
    const deadlineInput = document.getElementById('deadlineDate');
    if (deadlineInput) {
        deadlineInput.min = today;
        deadlineInput.required = false;
    }
}

function escapeHtml(str) {
    if (!str) return '';
    const string = String(str);
    return string
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

async function createRequest() {
    const title = document.getElementById('requestTitle')?.value.trim();
    if (!title) { alert('Введите название заявки'); return; }
    
    const deadlineDate = document.getElementById('deadlineDate')?.value;
    let deadline_at = null;
    
    if (deadlineDate) {
        const today = new Date().toISOString().split('T')[0];
        if (deadlineDate < today) {
            alert('Дедлайн не может быть раньше сегодняшней даты');
            return;
        }
        deadline_at = new Date(deadlineDate).toISOString();
    }
    
    const selectedWorks = [];
    document.querySelectorAll('#templatesListNew .work-item').forEach(item => {
        const checkbox = item.querySelector('.template-checkbox');
        if (checkbox && checkbox.checked) {
            const commentTextarea = item.querySelector('.work-comment');
            const executorSelect = item.querySelector('.work-executor');
            selectedWorks.push({
                work_id: parseInt(checkbox.value),
                comment: commentTextarea?.value.trim() || null,
                executor_id: executorSelect?.value ? parseInt(executorSelect.value) : null
            });
        }
    });
    
    if (selectedWorks.length === 0) { 
        alert('Выберите хотя бы одну работу'); 
        return; 
    }
    
    const contourId = document.getElementById('contourSelect')?.value;
    if (!contourId || contourId === '') { 
        alert('Выберите контур развертывания'); 
        return; 
    }
    
    try {
        const draft = await apiRequest('/requests', 'POST', { 
            title, 
            contour_id: parseInt(contourId),
            deadline_at: deadline_at
        });
        
        const result = await addTasksToRequest(draft.id, selectedWorks);
        
        if (result.success) {
            alert(`Заявка "${title}" создана`);
            document.getElementById('requestTitle').value = '';
            document.getElementById('deadlineDate').value = '';
            document.querySelectorAll('#templatesListNew .template-checkbox:checked').forEach(cb => cb.checked = false);
            document.querySelectorAll('#templatesListNew .work-comment').forEach(textarea => textarea.value = '');
            document.querySelectorAll('#templatesListNew .work-executor').forEach(select => select.value = '');
            await loadCustomerRequests();
            
            document.querySelector('#tabActive').style.display = 'block';
            document.querySelector('#tabNew').style.display = 'none';
            document.querySelector('.tab-btn[data-tab="active"]').classList.add('active');
            document.querySelector('.tab-btn[data-tab="new"]').classList.remove('active');
        }
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

async function submitRequest(requestId) {
    if (isSubmitting) return;
    
    const submitBtn = document.getElementById(`submit-btn-${requestId}`);
    const originalText = submitBtn?.innerText;
    
    isSubmitting = true;
    if (submitBtn) {
        submitBtn.disabled = true;
        submitBtn.innerText = 'Отправка...';
    }
    
    try {
        await apiRequest(`/requests/${requestId}/submit`, 'POST');
        alert('Заявка отправлена на исполнение');
        await loadCustomerRequests();
    } catch (error) {
        let errorMessage = error.message;
        if (errorMessage.includes('no executors available')) {
            alert('Нет доступных исполнителей. Создайте исполнителя в системе.');
        } else if (errorMessage.includes('deadline has already passed')) {
            alert('Дедлайн просрочен. Продлите срок выполнения перед отправкой.');
        } else if (errorMessage.includes('must have at least one task')) {
            alert('В заявке нет задач. Добавьте хотя бы одну работу.');
        } else if (errorMessage.includes('executor')) {
            alert('У всех задач должен быть назначен исполнитель');
        } else {
            alert('Ошибка отправки: ' + errorMessage);
        }
        await loadCustomerRequests();
    } finally {
        isSubmitting = false;
        if (submitBtn) {
            submitBtn.disabled = false;
            submitBtn.innerText = originalText;
        }
    }
}

async function editRequest(requestId) {
    const request = await apiRequest(`/requests/${requestId}`, 'GET');
    const selectedWorkIds = request.tasks.map(t => t.work_id);
    const works = await apiRequest('/works', 'GET');
    
    const worksHtml = works.map(w => `
        <label style="display:block; margin:10px 0;">
            <input type="checkbox" value="${w.id}" ${selectedWorkIds.includes(w.id) ? 'checked' : ''} class="edit-work-checkbox">
            <strong>${w.name}</strong> — ${w.description || ''} (${w.normative_hours} ч)
        </label>
    `).join('');
    
    const modalHtml = `
        <div id="editModal" class="modal" style="display:flex;">
            <div class="modal-content" style="max-width:500px;">
                <span class="close" onclick="closeEditModal()">&times;</span>
                <h3>Редактирование заявки ${requestId}</h3>
                <div id="editWorksList">${worksHtml}</div>
                <button onclick="saveEditRequest(${requestId})" class="btn-primary" style="margin-top:15px;">Сохранить изменения</button>
            </div>
        </div>
    `;
    
    document.body.insertAdjacentHTML('beforeend', modalHtml);
}

function closeEditModal() {
    const modal = document.getElementById('editModal');
    if (modal) modal.remove();
}

async function saveEditRequest(requestId) {
    const checkboxes = document.querySelectorAll('#editWorksList .edit-work-checkbox:checked');
    const selectedIds = Array.from(checkboxes).map(cb => parseInt(cb.value));
    const uniqueIds = [...new Set(selectedIds)];
    
    try {
        const currentRequest = await apiRequest(`/requests/${requestId}`, 'GET');
        const currentTaskIds = currentRequest.tasks.map(t => t.id);
        
        for (const taskId of currentTaskIds) {
            await apiRequest(`/requests/${requestId}/tasks/${taskId}`, 'DELETE');
        }
        
        if (uniqueIds.length > 0) {
            const tasksToAdd = uniqueIds.map(work_id => ({ work_id, comment: null, executor_id: null }));
            await apiRequest(`/requests/${requestId}/tasks`, 'POST', { tasks: tasksToAdd });
        }
        
        closeEditModal();
        await loadCustomerRequests();
        alert('Заявка обновлена');
    } catch (error) {
        alert('Ошибка обновления: ' + error.message);
        await loadCustomerRequests();
        closeEditModal();
    }
}

async function deleteRequest(requestId) {
    if (confirm('Удалить заявку? Это действие необратимо.')) {
        try {
            await apiRequest(`/requests/${requestId}`, 'DELETE');
            await loadCustomerRequests();
            alert('Заявка удалена');
        } catch (error) {
            alert('Ошибка: ' + error.message);
        }
    }
}

// ========== ИСПОЛНИТЕЛЬ ==========
function renderExecutorView() {
    renderActiveTasksForExecutor();
}

function applyContourFilter() {
    activeContourFilter = document.getElementById('executorContourFilter')?.value || null;
    renderActiveTasksForExecutor();
}

function applyDeadlineFilter() {
    activeDeadlineFilter = document.getElementById('executorDeadlineFilter')?.value || 'all';
    renderActiveTasksForExecutor();
}

function applyTaskStatusFilter() {
    activeTaskStatusFilter = document.getElementById('executorTaskStatusFilter')?.value || 'all';
    renderActiveTasksForExecutor();
}

function renderActiveTasksForExecutor() {
    const container = document.getElementById('executorTasks');
    if (!container) return;
    
    if (executorTasks.length === 0) {
        container.innerHTML = '<div class="empty-state">Нет задач</div>';
        return;
    }
    
    let filteredTasks = [...executorTasks];
    
    if (activeContourFilter && activeContourFilter !== 'all') {
        filteredTasks = filteredTasks.filter(t => t.contour === activeContourFilter);
    }
    
    if (activeDeadlineFilter && activeDeadlineFilter !== 'all' && deadlineFilterMap[activeDeadlineFilter]) {
        filteredTasks = filteredTasks.filter(t => {
            if (!t.deadline) return false;
            const deadlineDate = new Date(t.deadline);
            return deadlineFilterMap[activeDeadlineFilter].filter(deadlineDate);
        });
    }
    
    if (activeTaskStatusFilter && activeTaskStatusFilter !== 'all') {
        filteredTasks = filteredTasks.filter(t => t.status === activeTaskStatusFilter);
    }
    
    const requestsMap = new Map();
    filteredTasks.forEach(task => {
        if (!requestsMap.has(task.requestId)) {
            requestsMap.set(task.requestId, {
                requestId: task.requestId,
                title: task.requestTitle || 'Без названия',
                contour: task.contour,
                contour_description: task.contour_description || '',
                contour_id: task.contour_id,
                deadline: task.deadline,
                tasks: []
            });
        }
        requestsMap.get(task.requestId).tasks.push(task);
    });
    
    const requestsList = Array.from(requestsMap.values()).map(req => {
        const allCompleted = req.tasks.every(t => t.status === 'Завершено');
        const anyInProgress = req.tasks.some(t => t.status === 'В работе');
        let status = 'В работе';
        if (allCompleted) status = 'Завершено';
        else if (!anyInProgress) status = 'В планах';
        return { ...req, status };
    });
    
    const sortedRequests = [...requestsList].sort((a, b) => {
        if (a.status === 'Завершено' && b.status !== 'Завершено') return 1;
        if (a.status !== 'Завершено' && b.status === 'Завершено') return -1;
        if (a.deadline && b.deadline) {
            return new Date(a.deadline) - new Date(b.deadline);
        }
        if (a.deadline && !b.deadline) return -1;
        if (!a.deadline && b.deadline) return 1;
        return 0;
    });
    
    const contours = [...new Set(executorTasks.map(t => t.contour).filter(c => c && c !== '-'))];
    
    container.innerHTML = `
        <div class="filter-section" style="margin-bottom:20px; padding:12px; background:#f9fafb; border-radius:8px; display:flex; gap:16px; flex-wrap:wrap;">
            <div>
                <label style="font-weight:500;">Фильтр по контуру: </label>
                <select id="executorContourFilter" onchange="applyContourFilter()" style="padding:6px 12px; border-radius:6px; border:1px solid #d1d5db;">
                    <option value="all">Все контуры</option>
                    ${contours.map(c => `<option value="${c}" ${activeContourFilter === c ? 'selected' : ''}>${c}</option>`).join('')}
                </select>
            </div>
            <div>
                <label style="font-weight:500;">Фильтр по дедлайну: </label>
                <select id="executorDeadlineFilter" onchange="applyDeadlineFilter()" style="padding:6px 12px; border-radius:6px; border:1px solid #d1d5db;">
                    <option value="all">Все</option>
                    <option value="overdue" ${activeDeadlineFilter === 'overdue' ? 'selected' : ''}>Просрочено</option>
                    <option value="today" ${activeDeadlineFilter === 'today' ? 'selected' : ''}>Сегодня</option>
                    <option value="tomorrow" ${activeDeadlineFilter === 'tomorrow' ? 'selected' : ''}>Завтра</option>
                    <option value="this_week" ${activeDeadlineFilter === 'this_week' ? 'selected' : ''}>На этой неделе</option>
                    <option value="later" ${activeDeadlineFilter === 'later' ? 'selected' : ''}>Позже</option>
                </select>
            </div>
            <div>
                <label style="font-weight:500;">Фильтр по статусу: </label>
                <select id="executorTaskStatusFilter" onchange="applyTaskStatusFilter()" style="padding:6px 12px; border-radius:6px; border:1px solid #d1d5db;">
                    <option value="all">Все</option>
                    <option value="В планах" ${activeTaskStatusFilter === 'В планах' ? 'selected' : ''}>В планах</option>
                    <option value="В работе" ${activeTaskStatusFilter === 'В работе' ? 'selected' : ''}>В работе</option>
                    <option value="Завершено" ${activeTaskStatusFilter === 'Завершено' ? 'selected' : ''}>Завершено</option>
                </select>
            </div>
        </div>
        ${sortedRequests.map(req => `
            <div class="request-item" data-request-id="${req.requestId}">
                <div class="request-header" onclick="toggleExecutorRequestDetails(${req.requestId})" style="cursor:pointer;">
                    <div style="display:flex; justify-content:space-between; align-items:center; flex-wrap:wrap; gap:10px;">
                        <div>
                            <strong style="font-size: 16px;">${escapeHtml(req.title)}</strong>
                            <span class="request-title" style="font-size: 13px; color: #666;">Заявка №${req.requestId}</span>
                            ${req.deadline ? `<span class="deadline-badge ${new Date(req.deadline) < new Date() && req.status !== 'Завершено' ? 'deadline-overdue' : ''}">до ${new Date(req.deadline).toLocaleDateString()}</span>` : '<span class="deadline-badge" style="background:#e5e7eb;">дедлайн не указан</span>'}
                        </div>
                        <div>
                            <span>Контур: ${req.contour}</span>
                            <span class="status-badge 
                                ${req.status === 'Завершено' ? 'status-done' : 
                                    req.status === 'В работе' ? 'status-progress' : 
                                    req.status === 'В планах' ? 'status-planned' : ''}
                            ">${req.status}</span>
                        </div>
                    </div>
                    <div style="margin-top:5px; font-size:12px; color:#666;">
                        <span>Всего задач: ${req.tasks.length}</span>
                    </div>
                    <div class="expand-icon" style="font-size:12px; margin-top:5px;">
                        ▼ Развернуть
                    </div>
                </div>
                <div id="executor-request-details-${req.requestId}" class="request-details" style="display:none; margin-top:15px;">
                    <div class="request-info-block" style="background:#f0f0f0; padding:12px; border-radius:8px; margin-bottom:15px;">
                        <strong>Информация о заявке</strong><br>
                        <span>Название: ${escapeHtml(req.title)}</span><br>
                        <span>Дедлайн: ${req.deadline ? new Date(req.deadline).toLocaleDateString() : 'не указан'}</span><br>
                        <span>Контур: ${escapeHtml(req.contour)}${req.contour_description ? ` (${escapeHtml(req.contour_description)})` : ''}</span><br>
                        <span>Статус: ${req.status}</span>
                    </div>
                    <div class="tasks-list">
                        <h4>Задачи:</h4>
                        ${req.tasks.map(task => {
                            return `
                            <div class="task-card">
                                <div class="task-header">
                                    <span class="task-title">${escapeHtml(task.name)}</span>
                                    <span class="status-badge 
                                    ${task.status === 'В планах' ? 'status-planned' : 
                                        task.status === 'В работе' ? 'status-progress' : 
                                        task.status === 'Завершено' ? 'status-done' : ''}
                                ">${task.status}</span>
                                </div>
                                <div class="task-meta">
                                    <span>Время: ${task.hours} ч</span>
                                    ${task.description ? `<span style="margin-left:12px; font-size:11px; color:#888;">📝 ${escapeHtml(task.description.length > 80 ? task.description.substring(0, 80) + '...' : task.description)}</span>` : ''}
                                </div>
                                ${task.comment ? `
                                    <div class="task-comment" style="margin-top:8px; font-size:12px; color:#666; background:#fef3c7; padding:8px; border-radius:6px;">
                                        💬 Комментарий: ${escapeHtml(task.comment)}
                                    </div>
                                ` : ''}
                                <div style="margin-top:10px;">
                                    ${task.status !== 'Завершено' ? `
                                        <button onclick="openTaskStatusModal(${req.requestId}, ${task.id})" class="btn-primary" style="padding:4px 12px; font-size:12px;">Изменить статус</button>
                                    ` : ''}
                                    <button onclick="showTaskDetailForExecutor(${req.requestId}, ${task.id})" class="btn-outline" style="margin-left:8px; padding:4px 12px; font-size:12px;">Детали задачи</button>
                                </div>
                            </div>
                        `}).join('')}
                    </div>
                </div>
            </div>
        `).join('')}
    `;
}

function toggleExecutorRequestDetails(requestId) {
    const detailsDiv = document.getElementById(`executor-request-details-${requestId}`);
    if (detailsDiv) {
        const isVisible = detailsDiv.style.display === 'block';
        detailsDiv.style.display = isVisible ? 'none' : 'block';
        const expandIcon = document.querySelector(`.request-item[data-request-id="${requestId}"] .expand-icon`);
        if (expandIcon) {
            expandIcon.innerHTML = isVisible ? '▼ Развернуть' : '▲ Свернуть';
        }
    }
}

function showTaskDetailForExecutor(requestId, taskId) {
    const task = executorTasks.find(t => t.id === taskId);
    if (!task) return;
    
    const oldModal = document.getElementById('taskDetailModal');
    if (oldModal) oldModal.remove();
    
    const modalHtml = `
        <div id="taskDetailModal" class="modal" style="display: flex; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); z-index: 1000; align-items: center; justify-content: center;">
            <div class="modal-content" style="max-width: 500px; width: 90%; max-height: 80vh; background: white; border-radius: 12px; position: relative; display: flex; flex-direction: column;">
                <div style="padding: 20px; border-bottom: 1px solid #e5e7eb; display: flex; justify-content: space-between; align-items: center;">
                    <h3 style="margin: 0;">Детали задачи</h3>
                    <span class="close" onclick="closeTaskDetailModal()" style="font-size: 28px; cursor: pointer; color: #999; line-height: 1;">&times;</span>
                </div>
                <div style="padding: 20px; overflow-y: auto; flex: 1;">
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Название</label>
                        <div class="value">${escapeHtml(task.name)}</div>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Описание</label>
                        <div class="value" style="white-space: pre-wrap;">${escapeHtml(task.description || '—')}</div>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Контур</label>
                        <div class="value">${escapeHtml(task.contour)}</div>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Заявка</label>
                        <div class="value">${requestId}</div>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Время</label>
                        <div class="value">${task.hours} ч</div>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Статус</label>
                        <div class="value">${task.status}</div>
                    </div>
                    ${task.comment ? `
                        <div class="task-detail-field" style="margin-bottom: 12px;">
                            <label style="font-weight: 600; display: block; margin-bottom: 4px;">Комментарий заказчика</label>
                            <div class="value" style="background: #fef3c7; padding: 8px; border-radius: 6px; white-space: pre-wrap;">${escapeHtml(task.comment)}</div>
                        </div>
                    ` : '<div class="task-detail-field" style="margin-bottom: 12px;"><label style="font-weight: 600; display: block; margin-bottom: 4px;">Комментарий заказчика</label><div class="value">—</div></div>'}
                </div>
                <div style="padding: 20px; border-top: 1px solid #e5e7eb;">
                    <button onclick="closeTaskDetailModal()" class="btn-outline" style="width: 100%;">Закрыть</button>
                </div>
            </div>
        </div>
    `;
    
    document.body.insertAdjacentHTML('beforeend', modalHtml);
}

function closeTaskDetailModal() {
    const modal = document.getElementById('taskDetailModal');
    if (modal) modal.remove();
}

function openTaskStatusModal(requestId, taskId) {
    const task = executorTasks.find(t => t.id === taskId);
    if (!task) return;
    currentDetailTask = { requestId, taskId };
    
    let availableStatuses = [];
    if (task.status === 'В планах') {
        availableStatuses = [{ value: 'in_progress', label: 'В работу' }];
    } else if (task.status === 'В работе') {
        availableStatuses = [{ value: 'completed', label: 'Завершено' }];
    }
    
    if (availableStatuses.length === 0) return;
    
    const oldModal = document.getElementById('taskStatusModal');
    if (oldModal) oldModal.remove();
    
    const modalHtml = `
        <div id="taskStatusModal" class="modal" style="display: flex; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); z-index: 1000; align-items: center; justify-content: center;">
            <div class="modal-content" style="max-width: 400px; width: 90%; background: white; border-radius: 12px;">
                <div style="padding: 20px; border-bottom: 1px solid #e5e7eb; display: flex; justify-content: space-between; align-items: center;">
                    <h3 style="margin: 0;">Изменение статуса задачи</h3>
                    <span class="close" onclick="closeTaskStatusModal()" style="font-size: 28px; cursor: pointer; color: #999; line-height: 1;">&times;</span>
                </div>
                <div style="padding: 20px;">
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Название</label>
                        <div class="value">${escapeHtml(task.name)}</div>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Текущий статус</label>
                        <div class="value">${task.status}</div>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Новый статус</label>
                        <select id="task-status-select" style="width: 100%; padding: 8px; border-radius: 6px; border: 1px solid #d1d5db;">
                            ${availableStatuses.map(s => `<option value="${s.value}">${s.label}</option>`).join('')}
                        </select>
                    </div>
                    <div style="display:flex; gap:10px; margin-top:20px;">
                        <button onclick="updateTaskStatusFromModal()" class="btn-primary" style="flex:1;">Сохранить</button>
                        <button onclick="closeTaskStatusModal()" class="btn-outline" style="flex:1;">Отмена</button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    document.body.insertAdjacentHTML('beforeend', modalHtml);
}

function closeTaskStatusModal() {
    const modal = document.getElementById('taskStatusModal');
    if (modal) modal.remove();
    currentDetailTask = null;
}

async function updateTaskStatusFromModal() {
    if (!currentDetailTask) return;
    
    const select = document.getElementById('task-status-select');
    const newStatus = select.value;
    
    try {
        await apiRequest(`/tasks/${currentDetailTask.taskId}/status`, 'PUT', { status: newStatus });
        
        const task = executorTasks.find(t => t.id === currentDetailTask.taskId);
        if (task) {
            task.status = statusMapping[newStatus] || newStatus;
        }
        
        closeTaskStatusModal();
        await loadExecutorTasks();
        alert(`Статус обновлён на "${task.status}"`);
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

function showTaskDetailForCustomer(requestId, taskId) {
    const request = mockRequests.find(r => r.id === requestId);
    if (!request) return;
    const task = request.tasks.find(t => t.id === taskId);
    if (!task) return;
    
    const oldModal = document.getElementById('taskDetailModal');
    if (oldModal) oldModal.remove();
    
    const modalHtml = `
        <div id="taskDetailModal" class="modal" style="display: flex; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); z-index: 1000; align-items: center; justify-content: center;">
            <div class="modal-content" style="max-width: 500px; width: 90%; max-height: 80vh; background: white; border-radius: 12px; position: relative; display: flex; flex-direction: column;">
                <div style="padding: 20px; border-bottom: 1px solid #e5e7eb; display: flex; justify-content: space-between; align-items: center;">
                    <h3 style="margin: 0;">Детали задачи</h3>
                    <span class="close" onclick="closeTaskDetailModal()" style="font-size: 28px; cursor: pointer; color: #999; line-height: 1;">&times;</span>
                </div>
                <div style="padding: 20px; overflow-y: auto; flex: 1;">
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Название</label>
                        <div class="value">${escapeHtml(task.name)}</div>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Контур</label>
                        <div class="value">${escapeHtml(request.contour)}</div>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Заявка</label>
                        <div class="value">${requestId}</div>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Время</label>
                        <div class="value">${task.hours} ч</div>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Статус</label>
                        <div class="value">${task.status}</div>
                    </div>
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Исполнитель</label>
                        <div class="value">${task.executor_name || 'Не назначен'}</div>
                    </div>
                ${task.comment ? `
                    <div class="task-detail-field" style="margin-bottom: 12px;">
                        <label style="font-weight: 600; display: block; margin-bottom: 4px;">Комментарий</label>
                        <div class="value" style="background: #fef3c7; padding: 8px; border-radius: 6px; white-space: pre-wrap; word-wrap: break-word; overflow-wrap: break-word;">${escapeHtml(task.comment)}</div>
                    </div>
                ` : ''}
                </div>
                <div style="padding: 20px; border-top: 1px solid #e5e7eb;">
                    <button onclick="closeTaskDetailModal()" class="btn-outline" style="width: 100%;">Закрыть</button>
                </div>
            </div>
        </div>
    `;
    
    document.body.insertAdjacentHTML('beforeend', modalHtml);
}

function closeRequestDetailModalCustomer() {
    document.getElementById('requestDetailModalCustomer').style.display = 'none';
}

function closeRequestDetailModal() {
    document.getElementById('requestDetailModal').style.display = 'none';
}

// ========== ВСПОМОГАТЕЛЬНЫЕ ==========
function initTabs() {
    document.querySelectorAll('.tab-btn:not(.admin-tab)').forEach(btn => {
        btn.onclick = () => {
            const container = btn.closest('.card');
            container.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            const tab = btn.getAttribute('data-tab');
            if (currentUser.role === 'customer') {
                document.getElementById('tabActive').style.display = tab === 'active' ? 'block' : 'none';
                document.getElementById('tabNew').style.display = tab === 'new' ? 'block' : 'none';
                if (tab === 'new') renderNewRequestForm();
            }
        };
    });
}

async function restoreSession() {
    const token = localStorage.getItem('token');
    if (!token) return false;
    try {
        const userData = await apiRequest('/me', 'GET');
        if (userData) {
            currentUser = { id: userData.id, login: userData.email, fullname: userData.name || userData.email, role: userData.role };
            updateUIAfterLogin();
            return true;
        }
        return false;
    } catch (error) {
        clearToken();
        return false;
    }
}

// ========== ЗАПУСК ==========
restoreSession();

const loginForm = document.getElementById('loginForm');
if (loginForm) {
    loginForm.onsubmit = (e) => {
        e.preventDefault();
        login(document.getElementById('username').value, document.getElementById('password').value);
    };
}

const logoutBtn = document.getElementById('logoutBtn');
if (logoutBtn) logoutBtn.onclick = logout;
