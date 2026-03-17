// agit dashboard JavaScript

// Theme management
const STORAGE_KEY = 'agit-dashboard-theme';
const THEMES = {
    LIGHT: 'light',
    DARK: 'dark'
};

function getSystemTheme() {
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches 
        ? THEMES.DARK 
        : THEMES.LIGHT;
}

function getStoredTheme() {
    return localStorage.getItem(STORAGE_KEY);
}

function setTheme(theme) {
    const html = document.documentElement;
    const isDark = theme === THEMES.DARK;
    
    if (isDark) {
        html.classList.add('dark-theme');
    } else {
        html.classList.remove('dark-theme');
    }
    
    // Update icon
    const icon = document.getElementById('theme-icon');
    if (icon) {
        icon.textContent = isDark ? '☀️' : '🌙';
    }
    
    // Store preference
    localStorage.setItem(STORAGE_KEY, theme);
}

function toggleTheme() {
    const current = getStoredTheme() || getSystemTheme();
    const newTheme = current === THEMES.DARK ? THEMES.LIGHT : THEMES.DARK;
    setTheme(newTheme);
}

// Initialize theme
function initTheme() {
    const stored = getStoredTheme();
    const system = getSystemTheme();
    const initialTheme = stored || system;
    setTheme(initialTheme);
}

document.addEventListener('DOMContentLoaded', function() {
    // Initialize theme
    initTheme();
    
    // Theme toggle button
    const themeToggle = document.getElementById('theme-toggle');
    if (themeToggle) {
        themeToggle.addEventListener('click', toggleTheme);
    }

    // Copy commit hash on click
    document.querySelectorAll('.commit-hash').forEach(el => {
        el.addEventListener('click', function() {
            const text = this.textContent;
            navigator.clipboard.writeText(text).then(() => {
                const original = this.textContent;
                this.textContent = 'Copied!';
                setTimeout(() => this.textContent = original, 1500);
            });
        });
    });

    // Expand/collapse long test commands
    document.querySelectorAll('.test-command').forEach(el => {
        if (el.textContent.length > 80) {
            const short = el.textContent.substring(0, 80) + '…';
            const full = el.textContent;
            el.textContent = short;
            el.style.cursor = 'pointer';
            el.title = 'Click to expand';
            el.addEventListener('click', function() {
                if (this.textContent === short) {
                    this.textContent = full;
                    this.title = 'Click to collapse';
                } else {
                    this.textContent = short;
                    this.title = 'Click to expand';
                }
            });
        }
    });

    // Filter tasks by status (if we add filters later)
    const statusFilter = document.getElementById('status-filter');
    if (statusFilter) {
        statusFilter.addEventListener('change', function() {
            const selected = this.value;
            document.querySelectorAll('.task-row').forEach(row => {
                if (selected === 'all' || row.classList.contains(`status-${selected}`)) {
                    row.style.display = '';
                } else {
                    row.style.display = 'none';
                }
            });
        });
    }
});