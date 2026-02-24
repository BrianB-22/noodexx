// Noodexx Client-Side JavaScript - Phase 2
// Implements toast notifications, command palette, and UI helpers

console.log('Noodexx app.js loaded');

// ============================================================================
// Toast Notification System
// Requirements: 17.1-17.5
// ============================================================================

/**
 * Display a toast notification
 * @param {string} message - The message to display
 * @param {string} type - The toast type: 'success', 'error', 'info', or 'warning'
 * @param {number} duration - Duration in milliseconds (default: 5000)
 */
function showToast(message, type = 'info', duration = 5000) {
    const container = document.getElementById('toast-container');
    if (!container) {
        console.error('Toast container not found');
        return;
    }

    // Create toast element
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    
    // Toast icon based on type
    const icons = {
        success: '<svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"/></svg>',
        error: '<svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"/></svg>',
        info: '<svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"/></svg>',
        warning: '<svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"/></svg>'
    };

    toast.innerHTML = `
        <div class="toast-icon">${icons[type] || icons.info}</div>
        <div class="toast-message">${escapeHtml(message)}</div>
        <button class="toast-close" aria-label="Close notification">
            <svg width="16" height="16" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"/>
            </svg>
        </button>
    `;

    // Add to container
    container.appendChild(toast);

    // Trigger animation
    setTimeout(() => toast.classList.add('toast-show'), 10);

    // Close button handler
    const closeBtn = toast.querySelector('.toast-close');
    closeBtn.addEventListener('click', () => dismissToast(toast));

    // Auto-dismiss after duration
    setTimeout(() => dismissToast(toast), duration);
}

/**
 * Dismiss a toast notification
 * @param {HTMLElement} toast - The toast element to dismiss
 */
function dismissToast(toast) {
    if (!toast || !toast.parentElement) return;
    
    toast.classList.remove('toast-show');
    toast.classList.add('toast-hide');
    
    setTimeout(() => {
        if (toast.parentElement) {
            toast.parentElement.removeChild(toast);
        }
    }, 300);
}

/**
 * Escape HTML to prevent XSS
 * @param {string} text - Text to escape
 * @returns {string} Escaped text
 */
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// ============================================================================
// Command Palette
// Requirements: 18.1-18.5
// ============================================================================

let commandPaletteVisible = false;
let availableCommands = [];

/**
 * Initialize command palette
 */
function initCommandPalette() {
    // Load available commands
    loadCommands();

    // Keyboard shortcut: Cmd+K or Ctrl+K
    document.addEventListener('keydown', function(e) {
        // Check for Cmd+K (Mac) or Ctrl+K (Windows/Linux)
        if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
            e.preventDefault();
            toggleCommandPalette();
        }
        
        // Close on Escape
        if (e.key === 'Escape' && commandPaletteVisible) {
            closeCommandPalette();
        }
    });

    // Command input handler
    const commandInput = document.getElementById('command-input');
    if (commandInput) {
        commandInput.addEventListener('input', function(e) {
            filterCommands(e.target.value);
        });

        commandInput.addEventListener('keydown', function(e) {
            handleCommandNavigation(e);
        });
    }

    // Click backdrop to close
    const palette = document.getElementById('command-palette');
    if (palette) {
        palette.addEventListener('click', function(e) {
            if (e.target.classList.contains('command-palette-backdrop') || 
                e.target.id === 'command-palette') {
                closeCommandPalette();
            }
        });
    }
}

/**
 * Load available commands and skills
 */
function loadCommands() {
    // Navigation commands
    availableCommands = [
        { id: 'dashboard', label: 'Go to Dashboard', action: () => window.location.href = '/', icon: '🏠' },
        { id: 'chat', label: 'Go to Chat', action: () => window.location.href = '/chat', icon: '💬' },
        { id: 'library', label: 'Go to Library', action: () => window.location.href = '/library', icon: '📚' },
        { id: 'settings', label: 'Go to Settings', action: () => window.location.href = '/settings', icon: '⚙️' },
        { id: 'new-chat', label: 'New Chat', action: () => { window.location.href = '/chat?new=true'; }, icon: '✨' }
    ];

    // Load manual-trigger skills from API
    fetch('/api/skills')
        .then(response => response.json())
        .then(data => {
            const skills = data.skills || [];
            skills.forEach(skill => {
                if (skill.triggers && skill.triggers.includes('manual')) {
                    availableCommands.push({
                        id: `skill-${skill.name}`,
                        label: `Run: ${skill.name}`,
                        action: () => runSkill(skill.name),
                        icon: '🔧',
                        description: skill.description
                    });
                }
            });
        })
        .catch(err => console.error('Failed to load skills:', err));
}

/**
 * Toggle command palette visibility
 */
function toggleCommandPalette() {
    if (commandPaletteVisible) {
        closeCommandPalette();
    } else {
        openCommandPalette();
    }
}

/**
 * Open command palette
 */
function openCommandPalette() {
    const palette = document.getElementById('command-palette');
    const input = document.getElementById('command-input');
    
    if (!palette) return;
    
    palette.classList.remove('hidden');
    commandPaletteVisible = true;
    
    // Focus input
    if (input) {
        input.value = '';
        input.focus();
    }
    
    // Render all commands
    renderCommands(availableCommands);
}

/**
 * Close command palette
 */
function closeCommandPalette() {
    const palette = document.getElementById('command-palette');
    if (!palette) return;
    
    palette.classList.add('hidden');
    commandPaletteVisible = false;
}

/**
 * Filter commands by search query (fuzzy match)
 * @param {string} query - Search query
 */
function filterCommands(query) {
    if (!query.trim()) {
        renderCommands(availableCommands);
        return;
    }

    const lowerQuery = query.toLowerCase();
    const filtered = availableCommands.filter(cmd => {
        const label = cmd.label.toLowerCase();
        const description = (cmd.description || '').toLowerCase();
        
        // Simple fuzzy match: check if all query characters appear in order
        let queryIndex = 0;
        for (let i = 0; i < label.length && queryIndex < lowerQuery.length; i++) {
            if (label[i] === lowerQuery[queryIndex]) {
                queryIndex++;
            }
        }
        
        // Also check description
        if (queryIndex < lowerQuery.length && description) {
            for (let i = 0; i < description.length && queryIndex < lowerQuery.length; i++) {
                if (description[i] === lowerQuery[queryIndex]) {
                    queryIndex++;
                }
            }
        }
        
        return queryIndex === lowerQuery.length;
    });

    renderCommands(filtered);
}

/**
 * Render commands in the palette
 * @param {Array} commands - Commands to render
 */
function renderCommands(commands) {
    const list = document.getElementById('command-list');
    if (!list) return;

    if (commands.length === 0) {
        list.innerHTML = '<li class="command-item-empty">No commands found</li>';
        return;
    }

    list.innerHTML = commands.map((cmd, index) => `
        <li class="command-item ${index === 0 ? 'selected' : ''}" data-command-id="${cmd.id}">
            <span class="command-icon">${cmd.icon}</span>
            <div class="command-info">
                <div class="command-label">${escapeHtml(cmd.label)}</div>
                ${cmd.description ? `<div class="command-description">${escapeHtml(cmd.description)}</div>` : ''}
            </div>
        </li>
    `).join('');

    // Add click handlers
    list.querySelectorAll('.command-item').forEach(item => {
        item.addEventListener('click', function() {
            const cmdId = this.getAttribute('data-command-id');
            executeCommand(cmdId);
        });
    });
}

/**
 * Handle keyboard navigation in command palette
 * @param {KeyboardEvent} e - Keyboard event
 */
function handleCommandNavigation(e) {
    const list = document.getElementById('command-list');
    if (!list) return;

    const items = list.querySelectorAll('.command-item:not(.command-item-empty)');
    if (items.length === 0) return;

    const selected = list.querySelector('.command-item.selected');
    let currentIndex = Array.from(items).indexOf(selected);

    if (e.key === 'ArrowDown') {
        e.preventDefault();
        currentIndex = (currentIndex + 1) % items.length;
    } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        currentIndex = (currentIndex - 1 + items.length) % items.length;
    } else if (e.key === 'Enter') {
        e.preventDefault();
        if (selected) {
            const cmdId = selected.getAttribute('data-command-id');
            executeCommand(cmdId);
        }
        return;
    } else {
        return;
    }

    // Update selection
    items.forEach(item => item.classList.remove('selected'));
    items[currentIndex].classList.add('selected');
    items[currentIndex].scrollIntoView({ block: 'nearest' });
}

/**
 * Execute a command by ID
 * @param {string} commandId - Command ID
 */
function executeCommand(commandId) {
    const command = availableCommands.find(cmd => cmd.id === commandId);
    if (command && command.action) {
        closeCommandPalette();
        command.action();
    }
}

/**
 * Run a skill
 * @param {string} skillId - Skill ID
 */
function runSkill(skillId) {
    showToast('Running skill...', 'info');
    
    fetch(`/api/skills/${skillId}/run`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({})
    })
    .then(response => response.json())
    .then(result => {
        if (result.error) {
            showToast(`Skill failed: ${result.error}`, 'error');
        } else {
            showToast('Skill completed successfully', 'success');
            console.log('Skill result:', result);
        }
    })
    .catch(err => {
        showToast('Failed to run skill', 'error');
        console.error('Skill execution error:', err);
    });
}

// ============================================================================
// Sidebar Toggle (with localStorage persistence)
// Note: Basic toggle is already in base.html, this adds additional helpers
// ============================================================================

/**
 * Get sidebar collapsed state
 * @returns {boolean} True if sidebar is collapsed
 */
function isSidebarCollapsed() {
    return localStorage.getItem('sidebarCollapsed') === 'true';
}

/**
 * Set sidebar collapsed state
 * @param {boolean} collapsed - Whether sidebar should be collapsed
 */
function setSidebarCollapsed(collapsed) {
    const sidebar = document.getElementById('sidebar');
    if (!sidebar) return;
    
    if (collapsed) {
        sidebar.classList.add('collapsed');
    } else {
        sidebar.classList.remove('collapsed');
    }
    
    localStorage.setItem('sidebarCollapsed', collapsed);
}

// ============================================================================
// Markdown Rendering Helper
// Note: Server-side rendering is preferred, this is a placeholder for client-side needs
// ============================================================================

/**
 * Render markdown to HTML (placeholder - server-side rendering preferred)
 * @param {string} markdown - Markdown text
 * @returns {string} HTML string
 */
function renderMarkdown(markdown) {
    // This is a placeholder. In production, markdown should be rendered server-side
    // using goldmark for security and consistency.
    // This function exists for potential client-side needs.
    
    console.warn('Client-side markdown rendering not implemented. Use server-side rendering.');
    return escapeHtml(markdown);
}

// ============================================================================
// Initialization
// ============================================================================

document.addEventListener('DOMContentLoaded', function() {
    console.log('Initializing Noodexx client-side features...');
    
    // Initialize command palette
    initCommandPalette();
    
    console.log('Noodexx client-side initialization complete');
});

// Export functions for use in other scripts and templates
window.showToast = showToast;
window.dismissToast = dismissToast;
window.toggleCommandPalette = toggleCommandPalette;
window.isSidebarCollapsed = isSidebarCollapsed;
window.setSidebarCollapsed = setSidebarCollapsed;
window.renderMarkdown = renderMarkdown;
