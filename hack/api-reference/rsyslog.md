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
<code>filters</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Filters contain the filters that will be used to determine when to call
the rsyslog log forwarding action.</p>
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
bool
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
<h3 id="rsyslog-relp.extensions.gardener.cloud/v1alpha1.TLS">TLS
</h3>
<p>
(<em>Appears on:</em>
<a href="#rsyslog-relp.extensions.gardener.cloud/v1alpha1.RsyslogRelpConfig">RsyslogRelpConfig</a>)
</p>
<p>
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
</tbody>
</table>
<hr/>
<p><em>
Generated with <a href="https://github.com/ahmetb/gen-crd-api-reference-docs">gen-crd-api-reference-docs</a>
</em></p>
