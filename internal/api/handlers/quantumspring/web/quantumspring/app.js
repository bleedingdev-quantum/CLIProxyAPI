// QuantumSpring AI Proxy Metrics Dashboard
// JavaScript application for fetching and visualizing usage metrics

let tokensChart, modelChart, providerChart;

// Fetch metrics from the API
async function fetchMetrics() {
    try {
        const response = await fetch('/_qs/metrics');

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const data = await response.json();
        updateUI(data);
        clearError();
        updateStatus(true);
        return data;
    } catch (error) {
        console.error('Failed to fetch metrics:', error);
        showError(`Failed to fetch metrics: ${error.message}`);
        updateStatus(false);
        return null;
    }
}

// Update all UI elements with new data
function updateUI(data) {
    updateKPIs(data);
    updateCharts(data);
    updateLastUpdated();

    // Show content, hide loading
    document.getElementById('loading').style.display = 'none';
    document.getElementById('content').style.display = 'block';
}

// Update KPI cards
function updateKPIs(data) {
    const totals = data.totals || {};

    // Total Requests
    document.getElementById('total-requests').textContent = formatNumber(totals.requests || 0);

    // Total Tokens
    document.getElementById('total-tokens').textContent = formatTokens(totals.tokens || 0);

    // Success Rate
    const successRate = totals.success_rate || 0;
    document.getElementById('success-rate').textContent = successRate.toFixed(1) + '%';

    // Average Latency
    const avgLatency = totals.avg_latency_ms || 0;
    document.getElementById('avg-latency').textContent = avgLatency.toFixed(0);
}

// Update all charts
function updateCharts(data) {
    updateTokensChart(data.timeseries || []);
    updateModelChart(data.by_model || []);
    updateProviderChart(data.by_provider || []);
}

// Update tokens over time chart
function updateTokensChart(timeseries) {
    const ctx = document.getElementById('tokens-chart');

    if (tokensChart) {
        tokensChart.destroy();
    }

    const labels = timeseries.map(t => new Date(t.bucket_start));
    const tokenData = timeseries.map(t => t.tokens);
    const requestData = timeseries.map(t => t.requests);

    tokensChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [
                {
                    label: 'Tokens',
                    data: tokenData,
                    borderColor: '#3b82f6',
                    backgroundColor: 'rgba(59, 130, 246, 0.1)',
                    tension: 0.4,
                    fill: true,
                    yAxisID: 'y',
                },
                {
                    label: 'Requests',
                    data: requestData,
                    borderColor: '#8b5cf6',
                    backgroundColor: 'rgba(139, 92, 246, 0.1)',
                    tension: 0.4,
                    fill: true,
                    yAxisID: 'y1',
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            interaction: {
                mode: 'index',
                intersect: false,
            },
            plugins: {
                legend: {
                    labels: {
                        color: '#e6edf3'
                    }
                },
                tooltip: {
                    backgroundColor: '#161b22',
                    borderColor: '#30363d',
                    borderWidth: 1,
                    titleColor: '#e6edf3',
                    bodyColor: '#8b949e',
                }
            },
            scales: {
                x: {
                    type: 'time',
                    time: {
                        unit: 'hour',
                        displayFormats: {
                            hour: 'MMM d, HH:mm'
                        }
                    },
                    grid: {
                        color: '#30363d'
                    },
                    ticks: {
                        color: '#8b949e'
                    }
                },
                y: {
                    type: 'linear',
                    display: true,
                    position: 'left',
                    title: {
                        display: true,
                        text: 'Tokens',
                        color: '#e6edf3'
                    },
                    grid: {
                        color: '#30363d'
                    },
                    ticks: {
                        color: '#8b949e',
                        callback: function(value) {
                            return formatTokens(value);
                        }
                    }
                },
                y1: {
                    type: 'linear',
                    display: true,
                    position: 'right',
                    title: {
                        display: true,
                        text: 'Requests',
                        color: '#e6edf3'
                    },
                    grid: {
                        drawOnChartArea: false,
                    },
                    ticks: {
                        color: '#8b949e'
                    }
                }
            }
        }
    });
}

// Update model usage chart
function updateModelChart(byModel) {
    const ctx = document.getElementById('model-chart');

    if (modelChart) {
        modelChart.destroy();
    }

    if (byModel.length === 0) {
        return;
    }

    const labels = byModel.map(m => m.model);
    const data = byModel.map(m => m.tokens);

    const colors = [
        '#3b82f6',
        '#8b5cf6',
        '#ec4899',
        '#f59e0b',
        '#10b981',
        '#06b6d4',
        '#6366f1',
        '#84cc16'
    ];

    modelChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: labels,
            datasets: [{
                data: data,
                backgroundColor: colors.slice(0, labels.length),
                borderColor: '#161b22',
                borderWidth: 2,
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'right',
                    labels: {
                        color: '#e6edf3',
                        padding: 15,
                        font: {
                            size: 12
                        }
                    }
                },
                tooltip: {
                    backgroundColor: '#161b22',
                    borderColor: '#30363d',
                    borderWidth: 1,
                    titleColor: '#e6edf3',
                    bodyColor: '#8b949e',
                    callbacks: {
                        label: function(context) {
                            const label = context.label || '';
                            const value = context.parsed || 0;
                            const total = context.dataset.data.reduce((a, b) => a + b, 0);
                            const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : 0;
                            return `${label}: ${formatTokens(value)} (${percentage}%)`;
                        }
                    }
                }
            }
        }
    });
}

// Update provider usage chart
function updateProviderChart(byProvider) {
    const ctx = document.getElementById('provider-chart');

    if (providerChart) {
        providerChart.destroy();
    }

    if (byProvider.length === 0) {
        return;
    }

    const labels = byProvider.map(p => p.provider);
    const data = byProvider.map(p => p.requests);

    providerChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: labels,
            datasets: [{
                label: 'Requests',
                data: data,
                backgroundColor: '#3b82f6',
                borderColor: '#1d4ed8',
                borderWidth: 1,
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: false
                },
                tooltip: {
                    backgroundColor: '#161b22',
                    borderColor: '#30363d',
                    borderWidth: 1,
                    titleColor: '#e6edf3',
                    bodyColor: '#8b949e',
                }
            },
            scales: {
                x: {
                    grid: {
                        color: '#30363d'
                    },
                    ticks: {
                        color: '#8b949e'
                    }
                },
                y: {
                    grid: {
                        color: '#30363d'
                    },
                    ticks: {
                        color: '#8b949e'
                    },
                    beginAtZero: true
                }
            }
        }
    });
}

// Format numbers with commas
function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toLocaleString();
}

// Format tokens (millions/thousands)
function formatTokens(tokens) {
    if (tokens >= 1000000) {
        return (tokens / 1000000).toFixed(2) + 'M';
    } else if (tokens >= 1000) {
        return (tokens / 1000).toFixed(1) + 'K';
    }
    return tokens.toString();
}

// Update last updated timestamp
function updateLastUpdated() {
    const now = new Date();
    const formatted = now.toLocaleTimeString();
    document.getElementById('last-updated').textContent = formatted;
}

// Show error message
function showError(message) {
    const container = document.getElementById('error-container');
    container.innerHTML = `
        <div class="error">
            <strong>Error:</strong> ${message}
        </div>
    `;
}

// Clear error message
function clearError() {
    document.getElementById('error-container').innerHTML = '';
}

// Update status badge
function updateStatus(online) {
    const badge = document.getElementById('status-badge');
    if (online) {
        badge.textContent = 'Online';
        badge.className = 'status-badge online';
    } else {
        badge.textContent = 'Offline';
        badge.className = 'status-badge offline';
    }
}

// Initialize dashboard
async function init() {
    console.log('Initializing QuantumSpring Metrics Dashboard...');
    await fetchMetrics();

    // Auto-refresh every 30 seconds
    setInterval(fetchMetrics, 30000);
}

// Start the application
init();
