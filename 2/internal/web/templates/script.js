const API_BASE = '';

async function shortenURL() {
    const url = document.getElementById('urlInput').value;
    const custom = document.getElementById('customInput').value;
    const resultDiv = document.getElementById('result');

    if (!url) {
        resultDiv.innerHTML = '<div class="error">Please enter a URL</div>';
        return;
    }

    try {
        const response = await fetch('/shorten', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                url: url,
                custom_short: custom || undefined
            })
        });

        const data = await response.json();

        if (response.ok) {
            const shortURL = `${window.location.origin}/s/${data.short_url}`;
            resultDiv.innerHTML = `
                <div class="success">
                    <strong>✅ Short URL created!</strong><br>
                    <a href="${shortURL}" target="_blank" class="short-url">${shortURL}</a><br>
                    <small>Original: ${data.original_url}</small>
                </div>
            `;
            document.getElementById('shortenForm').reset();
            loadURLs();
        } else {
            resultDiv.innerHTML = `<div class="error">Error: ${data.error}</div>`;
        }
    } catch (error) {
        resultDiv.innerHTML = `<div class="error">Network error: ${error.message}</div>`;
    }
}

async function loadURLs() {
    const urlsList = document.getElementById('urlsList');

    try {
        const response = await fetch('/urls');
        const urls = await response.json();

        if (response.ok) {
            if (urls.length === 0) {
                urlsList.innerHTML = '<p>No URLs created yet.</p>';
                return;
            }

            urlsList.innerHTML = urls.map(url => `
                <div class="url-item">
                    <div class="url-info">
                        <a href="/s/${url.short_url}" target="_blank" class="short-url">
                            ${window.location.origin}/s/${url.short_url}
                        </a>
                        <div class="original-url">${url.original_url}</div>
                        <small>👆 Clicks: ${url.clicks} | 📅 Created: ${new Date(url.created_at).toLocaleDateString()}</small>
                    </div>
                </div>
            `).join('');
        } else {
            urlsList.innerHTML = '<div class="error">Error loading URLs</div>';
        }
    } catch (error) {
        urlsList.innerHTML = `<div class="error">Network error: ${error.message}</div>`;
    }
}

async function loadAnalytics() {
    const shortURL = document.getElementById('analyticsInput').value;
    const analyticsResult = document.getElementById('analyticsResult');

    if (!shortURL) {
        analyticsResult.innerHTML = '<div class="error">Please enter a short URL</div>';
        return;
    }

    try {
        const response = await fetch(`/analytics/${shortURL}`);
        const analytics = await response.json();

        if (response.ok) {
            analyticsResult.innerHTML = `
                <div class="analytics-section">
                    <h3>📊 Total Clicks: ${analytics.total_clicks}</h3>
                </div>
                
                <div class="analytics-section">
                    <h3>📈 Daily Stats (Last 30 days)</h3>
                    ${analytics.daily_stats.length > 0 ? 
                        analytics.daily_stats.map(stat => `
                            <div class="stat-item">
                                <span>${stat.date}</span>
                                <span>${stat.clicks} clicks</span>
                            </div>
                        `).join('') : 
                        '<p>No clicks recorded yet.</p>'
                    }
                </div>
                
                <div class="analytics-section">
                    <h3>🌐 User Agents</h3>
                    ${analytics.user_agents.length > 0 ? 
                        analytics.user_agents.map(ua => `
                            <div class="stat-item">
                                <span>${ua.user_agent || 'Unknown'}</span>
                                <span>${ua.clicks} clicks</span>
                            </div>
                        `).join('') : 
                        '<p>No user agent data available.</p>'
                    }
                </div>
            `;
        } else {
            analyticsResult.innerHTML = `<div class="error">Error: ${analytics.error}</div>`;
        }
    } catch (error) {
        analyticsResult.innerHTML = `<div class="error">Network error: ${error.message}</div>`;
    }
}

document.getElementById('shortenForm').addEventListener('submit', function(e) {
    e.preventDefault();
    shortenURL();
});

document.addEventListener('DOMContentLoaded', loadURLs);