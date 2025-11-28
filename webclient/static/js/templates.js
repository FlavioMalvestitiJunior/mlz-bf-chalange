// API Base URL
const API_BASE = '/api';

let editingTemplateId = null;

// Load templates
async function loadTemplates() {
    const container = document.getElementById('templatesTableContainer');

    try {
        const response = await fetch(`${API_BASE}/templates`);
        const templates = await response.json();

        if (!templates || templates.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <div class="empty-state-icon">📝</div>
                    <p>Nenhum template cadastrado</p>
                    <button class="btn btn-primary" onclick="openModal()" style="margin-top: 1rem;">
                        ➕ Criar Primeiro Template
                    </button>
                </div>
            `;
            return;
        }

        const tableHTML = `
            <div class="table-container">
                <table class="table">
                    <thead>
                        <tr>
                            <th>Nome</th>
                            <th>Modelo do Produto</th>
                            <th>SNS Topic</th>
                            <th>Status</th>
                            <th>Criado em</th>
                            <th>Ações</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${templates.map(template => `
                            <tr>
                                <td><strong>${escapeHtml(template.name)}</strong></td>
                                <td>${escapeHtml(template.product_model)}</td>
                                <td>${template.sns_topic_arn ? truncate(template.sns_topic_arn, 30) : '-'}</td>
                                <td>
                                    <span class="badge ${template.is_active ? 'badge-active' : 'badge-inactive'}">
                                        ${template.is_active ? '✓ Ativo' : '✗ Inativo'}
                                    </span>
                                </td>
                                <td>${formatDate(template.created_at)}</td>
                                <td>
                                    <button class="btn btn-secondary" onclick="editTemplate(${template.id})" 
                                            style="padding: 0.5rem 1rem; margin-right: 0.5rem;">
                                        ✏️ Editar
                                    </button>
                                    <button class="btn btn-danger" onclick="deleteTemplate(${template.id})" 
                                            style="padding: 0.5rem 1rem;">
                                        🗑️ Excluir
                                    </button>
                                </td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            </div>
        `;

        container.innerHTML = tableHTML;
    } catch (error) {
        console.error('Error loading templates:', error);
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-state-icon">⚠️</div>
                <p>Erro ao carregar templates</p>
            </div>
        `;
    }
}

// Open modal for create/edit
function openModal(templateId = null) {
    editingTemplateId = templateId;
    const modal = document.getElementById('templateModal');
    const form = document.getElementById('templateForm');

    form.reset();

    if (templateId) {
        document.getElementById('modalTitle').textContent = 'Editar Template';
        loadTemplateData(templateId);
    } else {
        document.getElementById('modalTitle').textContent = 'Novo Template';
        document.getElementById('isActive').checked = true;
    }

    modal.classList.add('active');
}

// Close modal
function closeModal() {
    const modal = document.getElementById('templateModal');
    modal.classList.remove('active');
    editingTemplateId = null;
}

// Load template data for editing
async function loadTemplateData(id) {
    try {
        const response = await fetch(`${API_BASE}/templates/${id}`);
        const template = await response.json();

        document.getElementById('templateId').value = template.id;
        document.getElementById('templateName').value = template.name;
        document.getElementById('productModel').value = template.product_model;
        document.getElementById('titleField').value = template.title_field;
        document.getElementById('descriptionField').value = template.description_field || '';
        document.getElementById('priceField').value = template.price_field;
        document.getElementById('discountField').value = template.discount_field || '';
        document.getElementById('detailsFields').value = template.details_fields || '';
        document.getElementById('messageSchema').value = formatJSON(template.message_schema);
        document.getElementById('snsTopicArn').value = template.sns_topic_arn || '';
        document.getElementById('isActive').checked = template.is_active;
    } catch (error) {
        console.error('Error loading template:', error);
        alert('Erro ao carregar template');
        closeModal();
    }
}

// Save template (create or update)
async function saveTemplate(event) {
    event.preventDefault();

    const id = document.getElementById('templateId').value;
    const name = document.getElementById('templateName').value;
    const productModel = document.getElementById('productModel').value;
    const titleField = document.getElementById('titleField').value;
    const descriptionField = document.getElementById('descriptionField').value || null;
    const priceField = document.getElementById('priceField').value;
    const discountField = document.getElementById('discountField').value || null;
    const detailsFields = document.getElementById('detailsFields').value || null;
    const messageSchema = document.getElementById('messageSchema').value;
    const snsTopicArn = document.getElementById('snsTopicArn').value || null;
    const isActive = document.getElementById('isActive').checked;

    // Validate JSON schema
    try {
        JSON.parse(messageSchema);
    } catch (error) {
        alert('Schema da mensagem inválido! Deve ser um JSON válido.');
        return;
    }

    // Validate details fields if provided
    if (detailsFields) {
        try {
            const parsed = JSON.parse(detailsFields);
            if (!Array.isArray(parsed)) {
                alert('Campos para busca deve ser um array JSON! Ex: ["brand", "category"]');
                return;
            }
        } catch (error) {
            alert('Campos para busca inválido! Deve ser um array JSON válido.');
            return;
        }
    }

    const data = {
        name,
        product_model: productModel,
        title_field: titleField,
        description_field: descriptionField,
        price_field: priceField,
        discount_field: discountField,
        details_fields: detailsFields,
        message_schema: messageSchema,
        sns_topic_arn: snsTopicArn,
        is_active: isActive
    };

    try {
        const url = id ? `${API_BASE}/templates/${id}` : `${API_BASE}/templates`;
        const method = id ? 'PUT' : 'POST';

        const response = await fetch(url, {
            method,
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });

        if (!response.ok) {
            throw new Error('Failed to save template');
        }

        closeModal();
        loadTemplates();

        // Show success message
        alert(id ? 'Template atualizado com sucesso!' : 'Template criado com sucesso!');
    } catch (error) {
        console.error('Error saving template:', error);
        alert('Erro ao salvar template');
    }
}

// Edit template
function editTemplate(id) {
    openModal(id);
}

// Delete template
async function deleteTemplate(id) {
    if (!confirm('Tem certeza que deseja excluir este template?')) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/templates/${id}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            throw new Error('Failed to delete template');
        }

        loadTemplates();
        alert('Template excluído com sucesso!');
    } catch (error) {
        console.error('Error deleting template:', error);
        alert('Erro ao excluir template');
    }
}

// Utility functions
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function truncate(text, length) {
    return text.length > length ? text.substring(0, length) + '...' : text;
}

function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('pt-BR', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

function formatJSON(jsonString) {
    try {
        const obj = JSON.parse(jsonString);
        return JSON.stringify(obj, null, 2);
    } catch {
        return jsonString;
    }
}

// Close modal when clicking outside
document.getElementById('templateModal')?.addEventListener('click', function (e) {
    if (e.target === this) {
        closeModal();
    }
});

// Initial load
loadTemplates();
