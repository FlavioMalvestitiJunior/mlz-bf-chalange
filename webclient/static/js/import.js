// Import Template Management
const API_BASE = '/api/import-templates';

let currentTemplate = null;
let jsonData = null;

// Load templates on page load
document.addEventListener('DOMContentLoaded', () => {
    loadTemplates();
    setupEventListeners();
});

function setupEventListeners() {
    document.getElementById('importForm').addEventListener('submit', handleSubmit);
    document.getElementById('testUrlBtn').addEventListener('click', testS3URL);
    document.getElementById('cancelBtn').addEventListener('click', resetForm);
}

async function loadTemplates() {
    try {
        const response = await fetch(API_BASE);
        const templates = await response.json();
        renderTemplates(templates);
    } catch (error) {
        console.error('Error loading templates:', error);
        alert('Erro ao carregar templates');
    }
}

function renderTemplates(templates) {
    const container = document.getElementById('templatesList');

    if (templates.length === 0) {
        container.innerHTML = '<p>Nenhum template encontrado. Crie um novo template acima.</p>';
        return;
    }

    container.innerHTML = templates.map(template => `
        <div class="template-card">
            <div class="template-info">
                <h3>${template.name}</h3>
                <p><strong>URL:</strong> ${template.s3_url}</p>
                <p><strong>Status:</strong> 
                    <span class="status-badge ${template.is_active ? 'status-active' : 'status-inactive'}">
                        ${template.is_active ? 'Ativo' : 'Inativo'}
                    </span>
                </p>
                ${template.last_run_at ? `<p><strong>Última execução:</strong> ${new Date(template.last_run_at).toLocaleString('pt-BR')}</p>` : ''}
            </div>
            <div class="template-actions">
                <button class="btn-toggle ${template.is_active ? '' : 'inactive'}" onclick="toggleTemplate(${template.id}, ${!template.is_active})">
                    ${template.is_active ? 'Desativar' : 'Ativar'}
                </button>
                <button class="btn-edit" onclick="editTemplate(${template.id})">Editar</button>
                <button class="btn-delete" onclick="deleteTemplate(${template.id})">Excluir</button>
            </div>
        </div>
    `).join('');
}

async function testS3URL() {
    const s3Url = document.getElementById('s3Url').value;

    if (!s3Url) {
        alert('Por favor, insira uma URL do S3');
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/test`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ s3_url: s3Url })
        });

        if (!response.ok) {
            throw new Error('Falha ao testar URL');
        }

        jsonData = await response.json();
        displayJSONPreview(jsonData);
        suggestMappings(jsonData);
    } catch (error) {
        console.error('Error testing S3 URL:', error);
        alert('Erro ao testar URL do S3. Verifique se a URL está correta e acessível.');
    }
}

function displayJSONPreview(data) {
    const container = document.getElementById('jsonPreviewContainer');
    const preview = document.getElementById('jsonPreview');

    // Show first item if array, otherwise show the object
    const sampleData = Array.isArray(data) ? data[0] : data;
    preview.textContent = JSON.stringify(sampleData, null, 2);
    container.classList.remove('hidden');
}

function suggestMappings(data) {
    // Get sample object
    const sample = Array.isArray(data) ? data[0] : data;

    // Common field mappings
    const mappings = {
        'ProductName': ['titulo', 'title', 'name', 'product_name', 'productName'],
        'Price': ['price', 'preco', 'valor', 'currentPrice'],
        'OriginalPrice': ['oldPrice', 'originalPrice', 'precoOriginal', 'preco_original'],
        'Details': ['details', 'description', 'descricao', 'detalhes'],
        'CashbackPercentage': ['percentCashback', 'cashback', 'cashbackPercent'],
        'Source': ['source', 'origem', 'provider', 'fornecedor']
    };

    // Try to auto-fill mappings
    Object.keys(mappings).forEach(field => {
        const input = document.getElementById(`map_${field}`);
        if (!input.value) { // Only auto-fill if empty
            const possibleKeys = mappings[field];
            for (const key of possibleKeys) {
                if (hasKey(sample, key)) {
                    input.value = key;
                    break;
                }
            }
        }
    });
}

function hasKey(obj, key) {
    if (typeof obj !== 'object' || obj === null) return false;
    return key in obj;
}

async function handleSubmit(e) {
    e.preventDefault();

    const templateId = document.getElementById('templateId').value;
    const name = document.getElementById('templateName').value;
    const s3Url = document.getElementById('s3Url').value;
    const isActive = document.getElementById('isActive').checked;

    // Build mapping schema
    const mappingSchema = {};
    ['ProductName', 'Price', 'OriginalPrice', 'Details', 'CashbackPercentage', 'Source'].forEach(field => {
        const value = document.getElementById(`map_${field}`).value;
        if (value) {
            mappingSchema[field] = value;
        }
    });

    if (!mappingSchema.ProductName) {
        alert('O campo ProductName é obrigatório');
        return;
    }

    const template = {
        name,
        s3_url: s3Url,
        mapping_schema: JSON.stringify(mappingSchema),
        is_active: isActive
    };

    try {
        let response;
        if (templateId) {
            // Update existing
            response = await fetch(`${API_BASE}/${templateId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(template)
            });
        } else {
            // Create new
            response = await fetch(API_BASE, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(template)
            });
        }

        if (!response.ok) {
            throw new Error('Falha ao salvar template');
        }

        alert('Template salvo com sucesso!');
        resetForm();
        loadTemplates();
    } catch (error) {
        console.error('Error saving template:', error);
        alert('Erro ao salvar template');
    }
}

async function editTemplate(id) {
    try {
        const response = await fetch(`${API_BASE}/${id}`);
        const template = await response.json();

        document.getElementById('formTitle').textContent = 'Editar Template';
        document.getElementById('templateId').value = template.id;
        document.getElementById('templateName').value = template.name;
        document.getElementById('s3Url').value = template.s3_url;
        document.getElementById('isActive').checked = template.is_active;

        // Parse and fill mapping
        const mapping = JSON.parse(template.mapping_schema);
        Object.keys(mapping).forEach(field => {
            const input = document.getElementById(`map_${field}`);
            if (input) {
                input.value = mapping[field];
            }
        });

        // Scroll to form
        document.getElementById('templateForm').scrollIntoView({ behavior: 'smooth' });
    } catch (error) {
        console.error('Error loading template:', error);
        alert('Erro ao carregar template');
    }
}

async function deleteTemplate(id) {
    if (!confirm('Tem certeza que deseja excluir este template?')) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/${id}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            throw new Error('Falha ao excluir template');
        }

        alert('Template excluído com sucesso!');
        loadTemplates();
    } catch (error) {
        console.error('Error deleting template:', error);
        alert('Erro ao excluir template');
    }
}

async function toggleTemplate(id, newStatus) {
    try {
        // First get the template
        const getResponse = await fetch(`${API_BASE}/${id}`);
        const template = await getResponse.json();

        // Update the is_active status
        template.is_active = newStatus;

        // Save it back
        const response = await fetch(`${API_BASE}/${id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(template)
        });

        if (!response.ok) {
            throw new Error('Falha ao atualizar status');
        }

        loadTemplates();
    } catch (error) {
        console.error('Error toggling template:', error);
        alert('Erro ao atualizar status do template');
    }
}

function resetForm() {
    document.getElementById('formTitle').textContent = 'Novo Template de Importação';
    document.getElementById('importForm').reset();
    document.getElementById('templateId').value = '';
    document.getElementById('jsonPreviewContainer').classList.add('hidden');
    jsonData = null;
}
