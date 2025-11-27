// Main JavaScript functionality
function toggleMenu() {
    const navLinks = document.getElementById('navLinks');
    navLinks.classList.toggle('show');
}

// Close mobile menu when clicking outside
document.addEventListener('click', function(event) {
    const navLinks = document.getElementById('navLinks');
    const menuBtn = document.querySelector('.mobile-menu-btn');
    
    if (!menuBtn.contains(event.target) && !navLinks.contains(event.target)) {
        navLinks.classList.remove('show');
    }
});

// Utility function to truncate text
function truncate(text, length) {
    return text.length > length ? text.substring(0, length) + '...' : text;
}

// Add truncate function to String prototype
String.prototype.truncate = function(length) {
    return this.length > length ? this.substring(0, length) + '...' : this;
};

// Initialize tooltips if needed
function initTooltips() {
    const tooltips = document.querySelectorAll('[data-tooltip]');
    tooltips.forEach(tooltip => {
        tooltip.addEventListener('mouseenter', showTooltip);
        tooltip.addEventListener('mouseleave', hideTooltip);
    });
}

function showTooltip(event) {
    const tooltipText = event.target.getAttribute('data-tooltip');
    const tooltip = document.createElement('div');
    tooltip.className = 'tooltip';
    tooltip.textContent = tooltipText;
    document.body.appendChild(tooltip);
    
    const rect = event.target.getBoundingClientRect();
    tooltip.style.left = rect.left + 'px';
    tooltip.style.top = (rect.top - tooltip.offsetHeight - 5) + 'px';
}

function hideTooltip() {
    const tooltip = document.querySelector('.tooltip');
    if (tooltip) {
        tooltip.remove();
    }
}

// Initialize when page loads
document.addEventListener('DOMContentLoaded', function() {
    initTooltips();
    
    // Add loading states to all buttons with async actions
    document.querySelectorAll('form').forEach(form => {
        form.addEventListener('submit', function() {
            const button = this.querySelector('button[type="submit"]');
            if (button) {
                button.disabled = true;
                const originalText = button.textContent;
                button.innerHTML = '<span class="loading"></span> Processing...';
                
                // Re-enable button if form submission fails
                setTimeout(() => {
                    button.disabled = false;
                    button.textContent = originalText;
                }, 5000);
            }
        });
    });
});

// Export functions for global access
window.toggleMenu = toggleMenu;
window.truncate = truncate;