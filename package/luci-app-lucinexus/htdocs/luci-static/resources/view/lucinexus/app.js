'use strict';
'require view';
'require fs';
'require ui';

return view.extend({
	render: function() {
		var container = E('div', { 'class': 'cbi-map' }, [
			E('h2', {}, _('LuciNexus - Autonomous OpenWrt Natural Language Assistant')),
			E('div', { 'class': 'cbi-map-descr' }, _('Control and diagnose your router using plain conversational English with automatic rollback safety.')),
			
			E('div', { 'class': 'cbi-section' }, [
				E('h3', {}, _('Assistant Command Console')),
				E('div', { 'class': 'cbi-section-node' }, [
					E('div', { 'class': 'cbi-value' }, [
						E('label', { 'class': 'cbi-value-title' }, _('Command Prompt')),
						E('div', { 'class': 'cbi-value-field' }, [
							E('input', {
								'id': 'prompt_input',
								'type': 'text',
								'class': 'cbi-input-text',
								'style': 'width: 100%; font-size: 1.1rem; padding: 8px;',
								'placeholder': 'e.g. Restart the wifi, or open port 8080 for web server'
							})
						])
					]),
					E('div', { 'class': 'cbi-value' }, [
						E('label', { 'class': 'cbi-value-title' }, _('Action Mode')),
						E('div', { 'class': 'cbi-value-field' }, [
							E('button', {
								'class': 'cbi-button cbi-button-action',
								'click': function() { runCommand(true); }
							}, _('Preview Plan (Dry Run)')),
							' ',
							E('button', {
								'class': 'cbi-button cbi-button-apply',
								'click': function() { runCommand(false); }
							}, _('Execute with Safety Guard'))
						])
					])
				])
			]),

			E('div', { 'class': 'cbi-section' }, [
				E('h3', {}, _('Custom AI Provider / cURL Auto-Import')),
				E('div', { 'class': 'cbi-section-descr' }, _('Paste a cURL command from your AI provider (e.g. DeepSeek, Groq, Ollama, OpenRouter) to automatically extract and apply configuration.')),
				E('div', { 'class': 'cbi-value' }, [
					E('label', { 'class': 'cbi-value-title' }, _('cURL Snippet')),
					E('div', { 'class': 'cbi-value-field' }, [
						E('textarea', {
							'id': 'curl_input',
							'class': 'cbi-input-textarea',
							'rows': 4,
							'style': 'width: 100%; font-family: monospace; font-size: 0.9rem;',
							'placeholder': 'curl https://api.deepseek.com/chat/completions \
  -H "Authorization: Bearer sk-..." \
  -d '{"model": "deepseek-chat"}''
						}),
						E('button', {
							'class': 'cbi-button cbi-button-action',
							'style': 'margin-top: 8px;',
							'click': function() { importCurl(); }
						}, _('Auto-Detect, Test & Apply'))
					])
				])
			]),

			E('div', { 'id': 'output_card', 'class': 'cbi-section', 'style': 'display: none;' }, [
				E('h3', {}, _('Execution Results & Diff Preview')),
				E('pre', { 'id': 'output_display', 'style': 'background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; overflow-x: auto;' }, '')
			])
		]);

		function runCommand(dryRun) {
			var prompt = document.getElementById('prompt_input').value;
			if (!prompt) {
				ui.addNotification(null, E('p', {}, _('Please enter a command prompt.')));
				return;
			}
			var outputCard = document.getElementById('output_card');
			var outputDisplay = document.getElementById('output_display');
			outputCard.style.display = 'block';
			outputDisplay.textContent = 'Analyzing request and executing plan...';

			fs.exec_direct('/usr/bin/lucinexus', [dryRun ? '-dry-run=true' : '-approve', prompt])
				.then(function(res) {
					outputDisplay.textContent = res;
				})
				.catch(function(err) {
					outputDisplay.textContent = 'Error: ' + err.message;
				});
		}

		function importCurl() {
			var snippet = document.getElementById('curl_input').value;
			if (!snippet) {
				ui.addNotification(null, E('p', {}, _('Please paste a cURL command.')));
				return;
			}
			ui.showIndicator('Verifying & Applying Endpoint...');
			fs.exec_direct('/usr/bin/lucinexus', ['--curl', snippet])
				.then(function(out) {
					ui.hideIndicator();
					ui.addNotification(null, E('p', {}, _('Custom AI Provider applied successfully!')), 'info');
				})
				.catch(function(err) {
					ui.hideIndicator();
					ui.addNotification(null, E('p', {}, _('Import failed: ') + err.message), 'error');
				});
		}

		return container;
	}
});
