<p>Packages:</p>
<ul>
<li>
<a href="#rsyslog-relp.extensions.gardener.cloud%2fv1alpha1">rsyslog-relp.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="rsyslog-relp.extensions.gardener.cloud/v1alpha1">rsyslog-relp.extensions.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains the Rsyslog Relp Shoot extension.</p>
</p>
Resource Types:
<ul><li>
<a href="#rsyslog-relp.extensions.gardener.cloud/v1alpha1.RsyslogRelpConfig">RsyslogRelpConfig</a>
</li></ul>
<h3 id="rsyslog-relp.extensions.gardener.cloud/v1alpha1.RsyslogRelpConfig">RsyslogRelpConfig
</h3>
<p>
<p>RsyslogRelpConfig configuration resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
rsyslog-relp.extensions.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>RsyslogRelpConfig</code></td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
string
</em>
</td>
<td>
<p>Target is the target server to connect to via relp.</p>
</td>
</tr>
<tr>
<td>
<code>port</code></br>
<em>
int
</em>
</td>
<td>
<p>Port is the TCP port to use when connecting to the target server.</p>
</td>
</tr>
<tr>
<td>
<code>loggingRules</code></br>
<em>
<a href="#rsyslog-relp.extensions.gardener.cloud/v1alpha1.LoggingRule">
[]LoggingRule
</a>
</em>
</td>
<td>
<p>LoggingRules contain a list of LoggingRules that are used to determine which logs are
sent to the target server by the the rsyslog relp action.</p>
</td>
</tr>
<tr>
<td>
<code>tls</code></br>
<em>
<a href="#rsyslog-relp.extensions.gardener.cloud/v1alpha1.TLS">
TLS
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLS hods the TLS config.</p>
</td>
</tr>
<tr>
<td>
<code>rebindInterval</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>RebindInterval is the rebind interval for the rsyslog relp action.</p>
</td>
</tr>
<tr>
<td>
<code>timeout</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>Timeout is the connection timeout for the rsyslog relp action.</p>
</td>
</tr>
<tr>
<td>
<code>resumeRetryCount</code></br>
<em>
int
</em>
</td>
<td>
<em>(Optional)</em>
<p>ResumeRetryCount is the resume retry count for the rsyslog relp action.</p>
</td>
</tr>
<tr>
<td>
<code>reportSuspensionContinuation</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>ReportSuspensionContinuation determines whether suspension continuation in the relp action
should be reported.</p>
</td>
</tr>
<tr>
<td>
<code>auditRulesConfig</code></br>
<em>
<a href="#rsyslog-relp.extensions.gardener.cloud/v1alpha1.AuditRulesConfig">
AuditRulesConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AuditRulesConfig contains the config for the audit rules to be deployed on the shoot nodes.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="rsyslog-relp.extensions.gardener.cloud/v1alpha1.AuditRulesConfig">AuditRulesConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#rsyslog-relp.extensions.gardener.cloud/v1alpha1.RsyslogRelpConfig">RsyslogRelpConfig</a>)
</p>
<p>
<p>AuditConfig contains options to configure the audit system.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enabled</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Enabled determines whether audit configuration is enabled or not. If it is enabled,
the audit rules on the system will be replaced.</p>
</td>
</tr>
<tr>
<td>
<code>configMapReferenceName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ConfigMapReferenceName is the name of the reference for the configmap containing
the audit rules to set on the shoot nodes. When this is not set, the following default
rules are specified: TODO: add ref to default rules</p>
</td>
</tr>
</tbody>
</table>
<h3 id="rsyslog-relp.extensions.gardener.cloud/v1alpha1.AuthMode">AuthMode
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#rsyslog-relp.extensions.gardener.cloud/v1alpha1.TLS">TLS</a>)
</p>
<p>
<p>AuthMode is the type of authentication mode that can be used for the rsyslog relp connection to the target server.</p>
</p>
<h3 id="rsyslog-relp.extensions.gardener.cloud/v1alpha1.LoggingRule">LoggingRule
</h3>
<p>
(<em>Appears on:</em>
<a href="#rsyslog-relp.extensions.gardener.cloud/v1alpha1.RsyslogRelpConfig">RsyslogRelpConfig</a>)
</p>
<p>
<p>LoggingRule contains options that determines which logs are sent to the target server.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>programNames</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ProgramNames are the names of the programs for which logs are sent to the target server.</p>
</td>
</tr>
<tr>
<td>
<code>severity</code></br>
<em>
int
</em>
</td>
<td>
<p>Severity determines which logs are sent to the target server based on their severity.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="rsyslog-relp.extensions.gardener.cloud/v1alpha1.TLS">TLS
</h3>
<p>
(<em>Appears on:</em>
<a href="#rsyslog-relp.extensions.gardener.cloud/v1alpha1.RsyslogRelpConfig">RsyslogRelpConfig</a>)
</p>
<p>
<p>TLS contains options for the tls connection to the target server.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enabled</code></br>
<em>
bool
</em>
</td>
<td>
<p>Enabled determines whether TLS encryption should be used for the connection
to the target server.</p>
</td>
</tr>
<tr>
<td>
<code>secretReferenceName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretReferenceName is the name of the reference for the secret
containing the certificates for the TLS connection when encryption is enabled.</p>
</td>
</tr>
<tr>
<td>
<code>permittedPeer</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>PermittedPeer is the name of the rsyslog relp permitted peer.
Only peers which have been listed in this parameter may be connected to.</p>
</td>
</tr>
<tr>
<td>
<code>authMode</code></br>
<em>
<a href="#rsyslog-relp.extensions.gardener.cloud/v1alpha1.AuthMode">
AuthMode
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AuthMode is the mode used for mutual authentication.
Possible values are &ldquo;fingerprint&rdquo; or &ldquo;name&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>tlsLib</code></br>
<em>
<a href="#rsyslog-relp.extensions.gardener.cloud/v1alpha1.TLSLib">
TLSLib
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLSLib specifies the tls library that will be used by librelp on the shoot nodes.
If the field is omitted, the librelp default is used.
Possible values are &ldquo;openssl&rdquo; or &ldquo;gnutls&rdquo;.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="rsyslog-relp.extensions.gardener.cloud/v1alpha1.TLSLib">TLSLib
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#rsyslog-relp.extensions.gardener.cloud/v1alpha1.TLS">TLS</a>)
</p>
<p>
<p>TLSLib is the tls library that is used by the librelp library on the shoot&rsquo;s nodes.</p>
</p>
<hr/>
<p><em>
Generated with <a href="https://github.com/ahmetb/gen-crd-api-reference-docs">gen-crd-api-reference-docs</a>
</em></p>
