// API Base URL
const API_BASE = '/api';

// Load dashboard data
async function loadData() {
    try {
        await Promise.all([
            loadStats(),
            loadActiveUsers()
        ]);
    } catch (error) {
        console.error('Error loading data:', error);
    }
}

// Load statistics
async function loadStats() {
    try {
        const response = await fetch(`${API_BASE}/stats`);
        const stats = await response.json();

        document.getElementById('activeUsers').textContent = stats.active_users || 0;
        document.getElementById('totalUsers').textContent = stats.total_users || 0;
        document.getElementById('totalWishlists').textContent = stats.total_wishlists || 0;
        document.getElementById('recentOffers').textContent = stats.recent_offers || 0;

        // Animate numbers
        animateValue('activeUsers', 0, stats.active_users || 0, 1000);
        animateValue('totalUsers', 0, stats.total_users || 0, 1000);
        animateValue('totalWishlists', 0, stats.total_wishlists || 0, 1000);
        animateValue('recentOffers', 0, stats.recent_offers || 0, 1000);
    } catch (error) {
        console.error('Error loading stats:', error);
    }
}

// Search users
let searchTimeout;
function searchUsers(query) {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
        loadActiveUsers(query);
    }, 300);
}

// Load active users (or search results)
async function loadActiveUsers(query = '') {
    const container = document.getElementById('usersTableContainer');

    try {
        const endpoint = query ? `${API_BASE}/users/search?q=${encodeURIComponent(query)}` : `${API_BASE}/users/active`;
        const response = await fetch(endpoint);
        const users = await response.json();

        if (!users || users.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <div class="empty-state-icon">👤</div>
                    <p>${query ? 'Nenhum usuário encontrado' : 'Nenhum usuário ativo nas últimas 24 horas'}</p>
                </div>
            `;
            return;
        }

        const tableHTML = `
            <div class="table-container">
                <table class="table">
                    <thead>
                        <tr>
                            <th>Telegram ID</th>
                            <th>Nome</th>
                            <th>Username</th>
                            <th>Listas</th>
                            <th>Última Atividade</th>
                            <th>Ações</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${users.map(user => `
                            <tr>
                                <td><strong>${user.telegram_id}</strong></td>
                                <td>${formatName(user)}</td>
                                <td>${user.username ? '@' + user.username : '-'}</td>
                                <td><span class="badge badge-active">${user.wishlists}</span></td>
                                <td>${formatDate(user.last_active)}</td>
                                <td class="actions-cell">
                                    <button class="btn-icon" onclick="showWishlist(${user.telegram_id})" title="Ver Lista de Desejos">📋</button>
                                    <button class="btn-icon" onclick="toggleBlacklist(${user.telegram_id}, ${user.is_blacklisted || false})" title="${user.is_blacklisted ? 'Remover da Blacklist' : 'Adicionar à Blacklist'}">
                                        ${user.is_blacklisted ? '✅' : '🚫'}
                                    </button>
                                    <button class="btn-icon btn-danger" onclick="deleteUser(${user.telegram_id})" title="Deletar Usuário">🗑️</button>
                                </td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            </div>
        `;

        container.innerHTML = tableHTML;
    } catch (error) {
        console.error('Error loading users:', error);
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">⚠️</div>
                <p>Erro ao carregar usuários</p>
            </div>
        `;
    }
}

// Show Wishlist Modal
async function showWishlist(userId) {
    const modal = document.getElementById('wishlistModal');
    const content = document.getElementById('wishlistContent');

    modal.style.display = 'block';
    content.innerHTML = '<div class="spinner"></div>';

    try {
        const response = await fetch(`${API_BASE}/users/${userId}/wishlist`);
        const wishlist = await response.json();

        if (!wishlist || wishlist.length === 0) {
            content.innerHTML = '<p class="text-center">A lista de desejos está vazia.</p>';
            return;
        }

        content.innerHTML = `
            <table class="table">
                <thead>
                    <tr>
                        <th>Produto</th>
                        <th>Preço Alvo</th>
                        <th>Desconto</th>
                        <th>Criado em</th>
                    </tr>
                </thead>
                <tbody>
                    ${wishlist.map(item => `
                        <tr>
                            <td>${item.product_name}</td>
                            <td>R$ ${item.target_price.toFixed(2)}</td>
                            <td>${item.discount_percentage}%</td>
                            <td>${new Date(item.created_at).toLocaleDateString()}</td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
    } catch (error) {
        content.innerHTML = '<p class="text-error">Erro ao carregar lista de desejos.</p>';
    }
}

// Close Modal
function closeModal() {
    document.getElementById('wishlistModal').style.display = 'none';
}

// Close modal when clicking outside
window.onclick = function (event) {
    const modal = document.getElementById('wishlistModal');
    if (event.target == modal) {
        modal.style.display = 'none';
    }
}

// Toggle Blacklist
async function toggleBlacklist(userId, isBlacklisted) {
    if (!confirm(isBlacklisted ? 'Remover usuário da blacklist?' : 'Adicionar usuário à blacklist?')) return;

    try {
        const method = isBlacklisted ? 'DELETE' : 'POST';
        await fetch(`${API_BASE}/users/${userId}/blacklist`, { method });
        loadActiveUsers(document.getElementById('userSearch').value);
    } catch (error) {
        alert('Erro ao atualizar blacklist');
    }
}

// Delete User
async function deleteUser(userId) {
    if (!confirm('Tem certeza que deseja deletar este usuário? Esta ação não pode ser desfeita.')) return;

    try {
        await fetch(`${API_BASE}/users/${userId}`, { method: 'DELETE' });
        loadActiveUsers(document.getElementById('userSearch').value);
    } catch (error) {
        alert('Erro ao deletar usuário');
    }
}

// Format user name
function formatName(user) {
    const parts = [];
    if (user.first_name) parts.push(user.first_name);
    if (user.last_name) parts.push(user.last_name);
    return parts.length > 0 ? parts.join(' ') : 'Sem nome';
}

// Format date
function formatDate(dateString) {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);

    if (diffMins < 1) return 'Agora';
    if (diffMins < 60) return `${diffMins}m atrás`;
    if (diffHours < 24) return `${diffHours}h atrás`;

    return date.toLocaleDateString('pt-BR', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// Animate number counting
function animateValue(id, start, end, duration) {
    const element = document.getElementById(id);
    const range = end - start;
    const increment = range / (duration / 16);
    let current = start;

    const timer = setInterval(() => {
        current += increment;
        if ((increment > 0 && current >= end) || (increment < 0 && current <= end)) {
            current = end;
            clearInterval(timer);
        }
        element.textContent = Math.floor(current);
    }, 16);
}

// Auto-refresh every 30 seconds
setInterval(loadData, 30000);

// Initial load
loadData();
