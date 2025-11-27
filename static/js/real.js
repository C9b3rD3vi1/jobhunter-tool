// Real functionality with live data
class RealJobHunter {
    constructor() {
        this.initEventListeners();
    }

    initEventListeners() {
        // Auto-refresh jobs every 5 minutes
        setInterval(() => this.refreshJobs(), 300000);
    }

    async startScraping() {
        const btn = event.target;
        const originalText = btn.textContent;
        
        btn.textContent = 'Scraping...';
        btn.disabled = true;

        try {
            const response = await fetch('/jobs/scrape', {
                method: 'GET'
            });
            
            const result = await response.json();
            
            if (result.status === 'success') {
                this.showAlert('Scraping started! Jobs will appear shortly.', 'success');
                // Refresh page after 10 seconds to show new jobs
                setTimeout(() => location.reload(), 10000);
            } else {
                throw new Error(result.message);
            }
        } catch (error) {
            this.showAlert('Scraping failed: ' + error.message, 'error');
        } finally {
            btn.textContent = originalText;
            btn.disabled = false;
        }
    }

    async analyzeJob(jobId, title, company, description) {
        const modal = document.getElementById('analysis-modal');
        if (!modal) {
            this.createAnalysisModal();
        }

        // Show loading state
        document.getElementById('analysis-results').innerHTML = `
            <div class="text-center py-8">
                <div class="loading loading-spinner loading-lg"></div>
                <p class="mt-4">Analyzing job fit with real AI...</p>
            </div>
        `;

        document.getElementById('analysis-modal').showModal();

        try {
            const response = await fetch('/analyze-skills', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    job_description: description,
                    user_skills: await this.getUserSkills()
                })
            });

            const analysis = await response.json();
            this.displayRealAnalysis(analysis, title, company);
        } catch (error) {
            document.getElementById('analysis-results').innerHTML = `
                <div class="alert alert-error">
                    Analysis failed: ${error.message}
                </div>
            `;
        }
    }

    async generateCoverLetter(jobTitle, company, jobDescription) {
        const userProfile = "Cybersecurity professional with experience in Fortinet, AWS security, SIEM, and cloud security. Strong background in SOC operations and incident response.";
        
        try {
            const response = await fetch('/cover-letter', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    job_title: jobTitle,
                    company: company,
                    job_description: jobDescription,
                    user_profile: userProfile
                })
            });

            const result = await response.json();
            this.showCoverLetter(result.cover_letter, jobTitle, company);
        } catch (error) {
            this.showAlert('Failed to generate cover letter', 'error');
        }
    }

    displayRealAnalysis(analysis, title, company) {
        const resultsDiv = document.getElementById('analysis-results');
        
        resultsDiv.innerHTML = `
            <div class="card bg-base-100 shadow">
                <div class="card-body">
                    <h3 class="card-title">Skills Fit Analysis: ${title} at ${company}</h3>
                    
                    <div class="flex items-center gap-4 mb-6">
                        <div class="radial-progress text-primary border-4 border-primary" 
                             style="--value:${analysis.fit_score};--size:8rem;">
                            ${analysis.fit_score}%
                        </div>
                        <div>
                            <h4 class="font-bold text-lg">Overall Fit Score</h4>
                            <p class="text-sm text-gray-600">Based on skill matching and requirements</p>
                        </div>
                    </div>
                    
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div class="bg-success/10 p-4 rounded-lg">
                            <h4 class="font-bold text-success mb-3">✅ Matching Skills (${analysis.matching_skills.length})</h4>
                            <div class="flex flex-wrap gap-2">
                                ${analysis.matching_skills.map(skill => 
                                    `<span class="badge badge-success">${skill}</span>`
                                ).join('')}
                            </div>
                        </div>
                        
                        <div class="bg-error/10 p-4 rounded-lg">
                            <h4 class="font-bold text-error mb-3">❌ Missing Skills (${analysis.missing_skills.length})</h4>
                            <div class="flex flex-wrap gap-2">
                                ${analysis.missing_skills.map(skill => 
                                    `<span class="badge badge-error">${skill}</span>`
                                ).join('')}
                            </div>
                        </div>
                    </div>
                    
                    ${analysis.recommendations && analysis.recommendations.length > 0 ? `
                        <div class="mt-6 bg-info/10 p-4 rounded-lg">
                            <h4 class="font-bold text-info mb-3">💡 AI Recommendations</h4>
                            <ul class="list-disc list-inside space-y-2">
                                ${analysis.recommendations.map(rec => 
                                    `<li>${rec}</li>`
                                ).join('')}
                            </ul>
                        </div>
                    ` : ''}
                    
                    <div class="card-actions justify-end mt-6">
                        <button class="btn btn-primary" onclick="jobhunter.generateCoverLetter('${title}', '${company}', \`${document.getElementById('job-description').value}\`)">
                            Generate Cover Letter
                        </button>
                        <button class="btn btn-outline" onclick="document.getElementById('analysis-modal').close()">
                            Close
                        </button>
                    </div>
                </div>
            </div>
        `;
    }

    createAnalysisModal() {
        const modalHTML = `
            <dialog id="analysis-modal" class="modal modal-bottom sm:modal-middle">
                <div class="modal-box max-w-4xl">
                    <form method="dialog">
                        <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2">✕</button>
                    </form>
                    <h3 class="font-bold text-lg mb-4">Job Fit Analysis</h3>
                    <div id="analysis-results"></div>
                </div>
            </dialog>
        `;
        document.body.insertAdjacentHTML('beforeend', modalHTML);
    }

    showCoverLetter(content, title, company) {
        const modalHTML = `
            <dialog id="cover-letter-modal" class="modal modal-bottom sm:modal-middle">
                <div class="modal-box max-w-4xl max-h-[80vh] overflow-y-auto">
                    <form method="dialog">
                        <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2">✕</button>
                    </form>
                    <h3 class="font-bold text-lg mb-4">Cover Letter: ${title} at ${company}</h3>
                    <div class="prose max-w-none">
                        <div class="bg-base-200 p-6 rounded-lg whitespace-pre-wrap">${content}</div>
                    </div>
                    <div class="modal-action">
                        <button class="btn btn-primary" onclick="navigator.clipboard.writeText(document.querySelector('#cover-letter-modal .bg-base-200').textContent)">
                            Copy to Clipboard
                        </button>
                        <button class="btn" onclick="document.getElementById('cover-letter-modal').close()">
                            Close
                        </button>
                    </div>
                </div>
            </dialog>
        `;
        
        // Remove existing modal
        const existingModal = document.getElementById('cover-letter-modal');
        if (existingModal) existingModal.remove();
        
        document.body.insertAdjacentHTML('beforeend', modalHTML);
        document.getElementById('cover-letter-modal').showModal();
    }

    showAlert(message, type = 'info') {
        const alertDiv = document.createElement('div');
        alertDiv.className = `alert alert-${type} fixed top-4 right-4 z-50 max-w-md`;
        alertDiv.innerHTML = `
            <span>${message}</span>
            <button class="btn btn-sm btn-ghost" onclick="this.parentElement.remove()">✕</button>
        `;
        document.body.appendChild(alertDiv);
        
        setTimeout(() => {
            if (alertDiv.parentElement) {
                alertDiv.remove();
            }
        }, 5000);
    }

    async getUserSkills() {
        // In a real implementation, this would fetch from user profile
        return ['AWS', 'Python', 'Go', 'Fortinet', 'SIEM', 'Docker', 'Kubernetes', 'Cybersecurity'];
    }

    async refreshJobs() {
        const response = await fetch('/jobs/scrape');
        if (response.ok) {
            this.showAlert('Jobs refreshed automatically', 'info');
        }
    }
}

// Initialize the application
const jobhunter = new RealJobHunter();

// Global functions for HTML onclick handlers
function startScraping() {
    jobhunter.startScraping();
}

function analyzeJob(jobId, title, company, description) {
    jobhunter.analyzeJob(jobId, title, company, description);
}