let currentUser = null;
let mockTemplates = [];
let mockRequests = [];
let executorTasks = [];
let currentDetailTask = null;
let adminUsers = [];
let adminContours = [];
let requestLogs = [];
let taskLogs = [];

const API_BASE = 'http://localhost:8080/api';

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
    'completed': 'Завершено'
};

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

function saveToLocalStorage() {
    localStorage.setItem('executorTasks', JSON.stringify(executorTasks));
}

function loadFromLocalStorage() {
    const savedTasks = localStorage.getItem('executorTasks');
    if (savedTasks) executorTasks = JSON.parse(savedTasks);
}

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
            select.innerHTML = contours.map(c => `<option value="${c.id}">${c.name}</option>`).join('');
        }
    } catch (error) {
        console.error('Ошибка загрузки контуров:', error);
    }
}

async function loadCustomerRequests() {
    try {
        const requests = await apiRequest('/requests', 'GET');
        mockRequests = requests.map(r => ({
            id: r.id,
            title: r.title || 'Без названия',
            contour: r.contour?.name || r.contour_name || '-',
            totalHours: r.total_hours || r.tasks?.reduce((sum, t) => sum + (t.work?.normative_hours || 1), 0) || 0,
            status: requestStatusMapping[r.status] || r.status,
            createdBy: r.customer_id,
            tasks: (r.tasks || []).map(t => ({
                id: t.id,
                name: t.work?.name || 'Без названия',
                hours: t.work?.normative_hours || 1,
                status: statusMapping[t.status] || t.status || 'В планах',
            }))
        }));
        renderCustomerView();
    } catch (error) {
        console.error('Ошибка загрузки заявок:', error);
        mockRequests = [];
        renderCustomerView();
    }
}

async function loadExecutorTasks() {
    try {
        const tasks = await apiRequest('/tasks', 'GET');
        console.log('Задачи из API:', tasks);
        
        executorTasks = tasks.map(t => ({
            id: t.id,
            name: t.work?.name || 'Без названия',
            hours: t.work?.normative_hours || 1,
            status: statusMapping[t.status] || t.status || 'В работе',
            requestId: t.request_id,
            contour: t.request?.contour?.name || '-',
        }));
        
        saveToLocalStorage();
        renderExecutorView();
        return executorTasks.filter(t => t.status !== 'Завершено');
    } catch (error) {
        console.error('Ошибка загрузки задач:', error);
        renderExecutorView();
        return [];
    }
}

// управление пользователями
async function loadAdminUsers() {
    try {
        const users = await apiRequest('/admin/users', 'GET');
        adminUsers = users;
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
    
    container.innerHTML = adminUsers.map(user => `
        <div class="user-card">
            <div class="user-info-text">
                <strong>${user.name || user.email}</strong><br>
                <small>Email: ${user.email} | Роль: ${user.role === 'customer' ? 'Заказчик' : user.role === 'executor' ? 'Исполнитель' : 'Админ'}</small>
            </div>
            <div class="user-actions">
                ${user.role !== 'admin' ? `
                    <button onclick="showEditUserModal(${user.id}, '${user.email}', '${user.name || ''}', '${user.role}')" class="btn-outline">Редактировать</button>
                    <button onclick="deleteAdminUser(${user.id})" class="btn-danger">Удалить</button>
                ` : ''}
            </div>
        </div>
    `).join('');
}

// создание изменение пользователей
function showCreateUserModal() {
    document.getElementById('newUserEmail').value = '';
    document.getElementById('newUserName').value = '';
    document.getElementById('newUserRole').value = 'customer';
    document.getElementById('newUserPassword').value = '';
    document.getElementById('createUserModal').style.display = 'flex';
}

function closeCreateUserModal() {
    document.getElementById('createUserModal').style.display = 'none';
}

async function createAdminUser() {
    const email = document.getElementById('newUserEmail').value.trim();
    const name = document.getElementById('newUserName').value.trim();
    const role = document.getElementById('newUserRole').value;
    const password = document.getElementById('newUserPassword').value;
    
    if (!email || !name || !password) {
        alert('Заполните все поля');
        return;
    }
    if (!email.includes('@')) {
        alert('Введите корректный email');
        return;
    }
    if (password.length < 3) {
        alert('Пароль должен быть не менее 3 символов');
        return;
    }
    
    try {
        await apiRequest('/admin/users', 'POST', { email, name, role, password });
        closeCreateUserModal();
        await loadAdminUsers();
        alert('Пользователь создан');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

function showEditUserModal(id, email, name, role) {
    document.getElementById('editUserId').value = id;
    document.getElementById('editUserEmail').value = email;
    document.getElementById('editUserName').value = name;
    document.getElementById('editUserRole').value = role;
    document.getElementById('editUserPassword').value = '';
    document.getElementById('editUserModal').style.display = 'flex';
}

function closeEditUserModal() {
    document.getElementById('editUserModal').style.display = 'none';
}

async function saveEditUser() {
    const id = document.getElementById('editUserId').value;
    const email = document.getElementById('editUserEmail').value.trim();
    const name = document.getElementById('editUserName').value.trim();
    const role = document.getElementById('editUserRole').value;
    const password = document.getElementById('editUserPassword').value;
    
    if (!email || !name) {
        alert('Заполните email и ФИО');
        return;
    }
    if (!email.includes('@')) {
        alert('Введите корректный email');
        return;
    }
    
    const body = { email, name, role };
    if (password && password.length >= 3) {
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
    if (!confirm('Удалить пользователя? Это действие необратимо.')) return;
    try {
        await apiRequest(`/admin/users/${userId}`, 'DELETE');
        await loadAdminUsers();
        alert('Пользователь удалён');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

// удалить контур
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
                <strong>${contour.name}</strong><br>
                <small>ID: ${contour.id}</small>
            </div>
            <div class="contour-actions">
                <button onclick="editAdminContour(${contour.id}, '${contour.name.replace(/'/g, "\\'")}')" class="btn-outline">Редактировать</button>
                <button onclick="deleteAdminContour(${contour.id})" class="btn-danger">Удалить</button>
            </div>
        </div>
    `).join('');
}

function showCreateContourModal() {
    document.getElementById('newContourName').value = '';
    document.getElementById('createContourModal').style.display = 'flex';
}

function closeCreateContourModal() {
    document.getElementById('createContourModal').style.display = 'none';
}

async function createAdminContour() {
    const name = document.getElementById('newContourName').value.trim();
    if (!name) {
        alert('Введите название контура');
        return;
    }
    try {
        await apiRequest('/admin/contours', 'POST', { name });
        closeCreateContourModal();
        await loadAdminContours();
        alert('Контур создан');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

function editAdminContour(id, oldName) {
    const newName = prompt('Введите название контура:', oldName);
    if (!newName || newName === oldName) return;
    updateAdminContour(id, newName);
}

async function updateAdminContour(id, name) {
    try {
        await apiRequest(`/admin/contours/${id}`, 'PUT', { name });
        await loadAdminContours();
        alert('Контур обновлён');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

async function deleteAdminContour(id) {
    if (!confirm('Удалить контур? Это действие необратимо.')) return;
    try {
        await apiRequest(`/admin/contours/${id}`, 'DELETE');
        await loadAdminContours();
        alert('Контур удалён');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

// журнал действий
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

// редактирование списка работ
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
    const hours = parseFloat(document.getElementById('editWorkHours').value);
    
    if (!name) { alert('Введите название работы'); return; }
    if (isNaN(hours) || hours < 1) { alert('Часы должны быть не менее 1'); return; }
    
    try {
        await apiRequest(`/admin/works/${id}`, 'PUT', { name, description, normative_hours: hours });
        closeEditWorkModal();
        await loadTemplatesFromServer();
        alert('Работа обновлена');
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

// регистрация
function showRegistrationModal() {
    document.getElementById('regEmail').value = '';
    document.getElementById('regFullname').value = '';
    document.getElementById('regPassword').value = '';
    document.getElementById('regConfirmPassword').value = '';
    document.getElementById('registrationModal').style.display = 'flex';
}

function closeRegistrationModal() {
    document.getElementById('registrationModal').style.display = 'none';
}

async function registerUser() {
    const email = document.getElementById('regEmail').value.trim();
    const fullname = document.getElementById('regFullname').value.trim();
    const role = document.getElementById('regRole').value;
    const password = document.getElementById('regPassword').value;
    const confirm = document.getElementById('regConfirmPassword').value;
    if (!email || !fullname || !password) { alert('Заполните все поля'); return; }
    if (password !== confirm) { alert('Пароли не совпадают'); return; }
    if (!email.includes('@')) { alert('Введите корректный email'); return; }
    try {
        const data = await apiRequest('/register', 'POST', { email, name: fullname, role, password }, false);
        if (data.token) {
            saveToken(data.token);
            currentUser = { id: data.user.id, login: data.user.email, fullname: data.user.name || data.user.email, role: data.user.role };
            closeRegistrationModal();
            updateUIAfterLogin();
            alert(`Регистрация успешна! Добро пожаловать, ${fullname}`);
        }
    } catch (error) {
        alert('Ошибка регистрации: ' + error.message);
    }
}

// авторизация
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
// после входа
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

// Админ
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
    const hours = parseFloat(document.getElementById('newTaskHours')?.value);
    if (!name || !desc) { alert('Заполните название и описание работы'); return; }
    if (isNaN(hours) || hours < 1) { alert('Нормативные часы должны быть не менее 1'); return; }
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

// Заказчик
function renderCustomerView() {
    renderActiveTasksForCustomer();
    renderNewRequestForm();
}

function renderActiveTasksForCustomer() {
    const container = document.getElementById('customerRequests');
    if (!container) return;
    
    if (mockRequests.length === 0) {
        container.innerHTML = '<div class="empty-state">Нет активных заявок</div>';
        return;
    }
    
    // вниз завершенные
    const sortedRequests = [...mockRequests].sort((a, b) => {
        if (a.status === 'Завершено' && b.status !== 'Завершено') return 1;
        if (a.status !== 'Завершено' && b.status === 'Завершено') return -1;
        return 0;
    });
    
    container.innerHTML = sortedRequests.map(req => `
        <div class="request-item" data-request-id="${req.id}">
            <div class="request-header" onclick="toggleRequestDetails(${req.id})" style="cursor:pointer;">
                <div style="display:flex; justify-content:space-between; align-items:center; flex-wrap:wrap; gap:10px;">
                    <div>
                        <strong>Заявка ${req.id}</strong>
                        <span class="request-title">${req.title || 'Без названия'}</span>
                    </div>
                    <div>
                        <span>Контур: ${req.contour}</span>
                        <span class="status-badge ${req.status === 'Завершено' ? 'status-done' : 'status-progress'}" style="margin-left:10px;">${req.status}</span>
                    </div>
                </div>
                <div style="margin-top:5px; font-size:12px; color:#666;">
                    <span>Общее время: ${req.totalHours} ч</span>
                </div>
                <div class="expand-icon" style="font-size:12px; margin-top:5px;">
                    ${req.status !== 'Завершено' ? '▼ Развернуть' : '▼ Просмотр'}
                </div>
            </div>
            <div id="request-details-${req.id}" class="request-details" style="display:none; margin-top:15px;">
                <div class="tasks-list">
                    <h4>Задачи: <br></h4>
                    ${req.tasks.map(task => `
                        <div class="task-card">
                            <div class="task-header">
                                <span class="task-title">${task.name}</span>
                                <span class="status-badge ${task.status === 'В планах' ? 'status-planned' : task.status === 'В работе' ? 'status-progress' : 'status-done'}">${task.status}</span>
                            </div>
                            <div class="task-meta">
                                <span>Время: ${task.hours} ч</span>
                            </div>
                            <div style="margin-top:10px;">
                                <button onclick="showTaskDetailForCustomer(${req.id}, ${task.id})" class="btn-outline" style="padding:4px 12px; font-size:12px;">Детали задачи</button>
                                ${req.status === 'Черновик' ? `
                                    <button onclick="deleteTaskFromDraft(${req.id}, ${task.id})" class="btn-danger" style="margin-left:8px; padding:4px 12px; font-size:12px;">Удалить задачу</button>
                                ` : ''}
                            </div>
                        </div>
                    `).join('')}
                </div>
                <div style="margin-top:15px;">
                    ${req.status === 'Черновик' ? `
                        <button onclick="editRequest(${req.id})" class="btn-outline" style="margin-right:10px;">Редактировать</button>
                        <button onclick="deleteRequest(${req.id})" class="btn-danger" style="margin-right:10px;">Удалить</button>
                        <button onclick="submitRequest(${req.id})" class="btn-primary">Отправить</button>
                    ` : ''}
                    <button onclick="showRequestDetail(${req.id})" class="btn-outline">Детали заявки</button>
                </div>
            </div>
        </div>
    `).join('');
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
    container.innerHTML = mockTemplates.map(t => `
<label style="display:flex; align-items:center; gap:12px; margin:12px 0; padding:10px; background:#f9fafb; border-radius:8px;">
    <input type="checkbox" value="${t.id}" class="template-checkbox">
    <div><strong>${t.name}</strong><br><small>${t.description} — ${t.hours} ч</small></div>
</label>
    `).join('');
}

async function createRequest() {
    const title = document.getElementById('requestTitle')?.value.trim();
    if (!title) { alert('Введите название заявки'); return; }
    const checkboxes = document.querySelectorAll('#templatesListNew .template-checkbox:checked');
    const selectedIds = Array.from(checkboxes).map(cb => parseInt(cb.value));
    if (selectedIds.length === 0) { alert('Выберите хотя бы одну работу'); return; }
    const contourId = document.getElementById('contourSelect')?.value;
    if (!contourId || contourId === '') { alert('Выберите контур развертывания'); return; }
    try {
        const draft = await apiRequest('/requests', 'POST', { title, contour_id: parseInt(contourId) });
        await apiRequest(`/requests/${draft.id}/tasks`, 'POST', { work_ids: selectedIds });
        alert(`Заявка "${title}" создана в статусе Черновик`);
        document.getElementById('requestTitle').value = '';
        document.querySelectorAll('#templatesListNew .template-checkbox:checked').forEach(cb => cb.checked = false);
        await loadCustomerRequests();
    } catch (error) {
        alert('Ошибка: ' + error.message);
    }
}

async function showRequestDetail(requestId) {
    try {
        const request = await apiRequest(`/requests/${requestId}`, 'GET');
        const tasks = request.tasks || [];
        
        let tasksHtml = '';
        for (const task of tasks) {
            const taskName = task.work?.name || task.name || 'Задача';
            const taskHours = task.work?.normative_hours || 0;
            const taskStatus = task.status === 'completed' ? 'Завершено' : (task.status === 'in_progress' ? 'В работе' : 'В планах');
            tasksHtml += `
                <div class="task-detail-item" style="padding: 8px; border-bottom: 1px solid #e0e0e0;">
                    <strong>${taskName}</strong><br>
                    <small>Статус: ${taskStatus} | Трудоёмкость: ${taskHours} ч</small>
                </div>
            `;
        }
        
        const statusText = request.status === 'completed' ? 'Завершено' : 
                          (request.status === 'submitted' ? 'Отправлена' : 
                          (request.status === 'in_progress' ? 'В работе' : 'Черновик'));
        
        const contourName = request.contour?.name || request.contour_name || '-';
        
        const content = `
            <div class="task-detail-field">
                <label>Название</label>
                <div class="value">${request.title || '-'}</div>
            </div>
            <div class="task-detail-field">
                <label>Контур</label>
                <div class="value">${contourName}</div>
            </div>
            <div class="task-detail-field">
                <label>Статус</label>
                <div class="value">${statusText}</div>
            </div>
            <div class="task-detail-field">
                <label>Общее время</label>
                <div class="value">${request.total_hours || 0} ч</div>
            </div>
            <div class="task-detail-field">
                <label>Создана</label>
                <div class="value">${new Date(request.created_at).toLocaleString()}</div>
            </div>
            <div class="task-detail-field">
                <label>Задачи</label>
                <div class="value" style="max-height: 200px; overflow-y: auto;">
                    ${tasksHtml || '— нет —'}
                </div>
            </div>
        `;
        
        document.getElementById('requestDetailContentCustomer').innerHTML = content;
        document.getElementById('requestDetailModalCustomer').style.display = 'flex';
    } catch (error) {
        alert('Ошибка загрузки деталей заявки: ' + error.message);
    }
}

function closeRequestDetailModalCustomer() {
    document.getElementById('requestDetailModalCustomer').style.display = 'none';
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
    const currentRequest = await apiRequest(`/requests/${requestId}`, 'GET');
    const currentTaskIds = currentRequest.tasks.map(t => t.id);
    for (const taskId of currentTaskIds) {
        await apiRequest(`/requests/${requestId}/tasks/${taskId}`, 'DELETE');
    }
    if (selectedIds.length > 0) {
        await apiRequest(`/requests/${requestId}/tasks`, 'POST', { work_ids: selectedIds });
    }
    closeEditModal();
    await loadCustomerRequests();
    alert('Заявка обновлена');
}

async function submitRequest(requestId) {
    if (confirm('Отправить заявку на исполнение? После отправки редактирование будет недоступно.')) {
        try {
            await apiRequest(`/requests/${requestId}/submit`, 'POST');
            await loadCustomerRequests();
            alert('Заявка отправлена на исполнение');
        } catch (error) {
            alert('Ошибка: ' + error.message);
        }
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

async function deleteTaskFromDraft(requestId, taskId) {
    if (confirm('Удалить задачу из заявки?')) {
        try {
            await apiRequest(`/requests/${requestId}/tasks/${taskId}`, 'DELETE');
            await loadCustomerRequests();
            alert('Задача удалена');
        } catch (error) {
            alert('Ошибка: ' + error.message);
        }
    }
}

//Исполнитель
function renderExecutorView() {
    renderActiveTasksForExecutor();
}

function renderActiveTasksForExecutor() {
    const container = document.getElementById('executorTasks');
    if (!container) return;
    
    if (executorTasks.length === 0) {
        container.innerHTML = '<div class="empty-state">Нет задач</div>';
        return;
    }
    
    // убрать вниз завершенные
    const sortedTasks = [...executorTasks].sort((a, b) => {
        if (a.status === 'Завершено' && b.status !== 'Завершено') return 1;
        if (a.status !== 'Завершено' && b.status === 'Завершено') return -1;
        return 0;
    });
    
    container.innerHTML = sortedTasks.map(task => `
        <div class="task-card ${task.status === 'Завершено' ? 'task-completed' : ''}" data-task-id="${task.id}">
            <div class="task-header">
                <span class="task-title">${task.name}</span>
                <span class="status-badge ${task.status === 'В планах' ? 'status-planned' : task.status === 'В работе' ? 'status-progress' : 'status-done'}">${task.status}</span>
            </div>
            <div class="task-meta">
                <span>Время: ${task.hours} ч</span>
                <span>Контур: ${task.contour}</span>
                <span>Заявка: ${task.requestId}</span>
            </div>
            <div style="margin-top:12px; display:flex; gap:10px;">
                ${task.status !== 'Завершено' ? `
                    <button onclick="openTaskStatusModal(${task.requestId}, ${task.id})" class="btn-primary">Изменить статус</button>
                ` : ''}
                <button onclick="showRequestDetailForExecutor(${task.requestId})" class="btn-outline">Детали заявки</button>
            </div>
        </div>
    `).join('');
}

function openTaskStatusModal(requestId, taskId) {
    const task = executorTasks.find(t => t.id === taskId);
    if (!task) return;
    currentDetailTask = { requestId, taskId };
    
    //статусы: в планах -> в работе, в работе -> завершено
    let availableStatuses = [];
    if (task.status === 'В планах') {
        availableStatuses = [{ value: 'in_progress', label: 'В работе' }];
    } else if (task.status === 'В работе') {
        availableStatuses = [{ value: 'completed', label: 'Завершено' }];
    }
    
    if (availableStatuses.length === 0) return;
    
    const modalHtml = `
        <div class="modal task-detail-modal" id="taskStatusModal" style="display:flex;">
            <div class="modal-content">
                <span class="close" onclick="closeTaskStatusModal()">&times;</span>
                <h3>Изменение статуса задачи</h3>
                <div class="task-detail-field">
                    <label>Название</label>
                    <div class="value">${task.name}</div>
                </div>
                <div class="task-detail-field">
                    <label>Текущий статус</label>
                    <div class="value">${task.status}</div>
                </div>
                <div class="task-detail-field">
                    <label>Новый статус</label>
                    <select id="task-status-select">
                        ${availableStatuses.map(s => `<option value="${s.value}">${s.label}</option>`).join('')}
                    </select>
                </div>
                <div style="display:flex; gap:10px; margin-top:20px;">
                    <button onclick="updateTaskStatusFromModal()" class="btn-primary">Сохранить</button>
                    <button onclick="closeTaskStatusModal()" class="btn-outline">Отмена</button>
                </div>
            </div>
        </div>
    `;
    
    const oldModal = document.getElementById('taskStatusModal');
    if (oldModal) oldModal.remove();
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
    const backendStatus = reverseStatusMapping[newStatus === 'in_progress' ? 'В работе' : (newStatus === 'completed' ? 'Завершено' : newStatus)];
    
    try {
        await apiRequest(`/tasks/${currentDetailTask.taskId}/status`, 'PUT', { status: newStatus });
        
        // Обновляем локальные данные
        const task = executorTasks.find(t => t.id === currentDetailTask.taskId);
        if (task) {
            task.status = statusMapping[newStatus] || newStatus;
        }
        
        saveToLocalStorage();
        closeTaskStatusModal();
        renderExecutorView();
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
    currentDetailTask = { requestId, taskId, role: 'customer' };
    
    const modalHtml = `
        <div class="modal task-detail-modal" id="taskDetailModal" style="display:flex;">
            <div class="modal-content">
                <span class="close" onclick="closeTaskDetailModal()">&times;</span>
                <h3>Детали задачи</h3>
                <div class="task-detail-field"><label>Название</label><div class="value">${task.name}</div></div>
                <div class="task-detail-field"><label>Контур</label><div class="value">${request.contour}</div></div>
                <div class="task-detail-field"><label>Заявка</label><div class="value">${requestId}</div></div>
                <div class="task-detail-field"><label>Время</label><div class="value">${task.hours} ч</div></div>
                <div class="task-detail-field"><label>Статус</label><div class="value">${task.status}</div></div>
                <button onclick="closeTaskDetailModal()" class="btn-outline">Закрыть</button>
            </div>
        </div>
    `;
    const oldModal = document.getElementById('taskDetailModal');
    if (oldModal) oldModal.remove();
    document.body.insertAdjacentHTML('beforeend', modalHtml);
}

function closeTaskDetailModal() {
    const modal = document.getElementById('taskDetailModal');
    if (modal) modal.remove();
    currentDetailTask = null;
}

async function showRequestDetailForExecutor(requestId) {
    try {
        const request = await apiRequest(`/requests/${requestId}`, 'GET');
        const tasks = request.tasks || [];
        
        let tasksHtml = '';
        for (const task of tasks) {
            const taskName = task.work?.name || task.name || 'Задача';
            const taskHours = task.work?.normative_hours || 0;
            const taskStatus = task.status === 'completed' ? 'Завершено' : (task.status === 'in_progress' ? 'В работе' : 'В планах');
            tasksHtml += `
                <div class="task-detail-item">
                    <strong>${taskName}</strong><br>
                    <small>Статус: ${taskStatus} | Трудоёмкость: ${taskHours} ч</small>
                </div>
            `;
        }
        
        const statusText = request.status === 'completed' ? 'Завершено' : 
                          (request.status === 'submitted' ? 'Отправлена' : 
                          (request.status === 'in_progress' ? 'В работе' : 'Черновик'));
        
        const contourName = request.contour?.name || request.contour_name || '-';
        
        const content = `
            <div class="task-detail-field">
                <label>Название</label>
                <div class="value">${request.title || '-'}</div>
            </div>
            <div class="task-detail-field">
                <label>Контур</label>
                <div class="value">${contourName}</div>
            </div>
            <div class="task-detail-field">
                <label>Статус</label>
                <div class="value">${statusText}</div>
            </div>
            <div class="task-detail-field">
                <label>Общее время</label>
                <div class="value">${request.total_hours || 0} ч</div>
            </div>
            <div class="task-detail-field">
                <label>Создана</label>
                <div class="value">${new Date(request.created_at).toLocaleString()}</div>
            </div>
            <div class="task-detail-field">
                <label>Задачи</label>
                <div class="value" style="max-height: 200px; overflow-y: auto;">
                    ${tasksHtml || '— нет —'}
                </div>
            </div>
        `;
        
        document.getElementById('requestDetailContent').innerHTML = content;
        document.getElementById('requestDetailModal').style.display = 'flex';
    } catch (error) {
        alert('Ошибка загрузки деталей заявки: ' + error.message);
    }
}

function closeRequestDetailModal() {
    document.getElementById('requestDetailModal').style.display = 'none';
}


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
//восстановление при обновлении 
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

//запуск
loadFromLocalStorage();
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