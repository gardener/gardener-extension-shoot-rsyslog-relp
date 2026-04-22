<p>Packages:</p>
<ul>
<li>
<a href="#rsyslog-relp.extensions.gardener.cloud%2fv1alpha1">rsyslog-relp.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>

<h2 id="rsyslog-relp.extensions.gardener.cloud/v1alpha1">rsyslog-relp.extensions.gardener.cloud/v1alpha1</h2>
<p>

</p>

<h3 id="auditconfig">AuditConfig
</h3>


<p>
(<em>Appears on:</em><a href="#rsyslogrelpconfig">RsyslogRelpConfig</a>)
</p>

<p>
AuditConfig contains options to configure the audit system.
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
boolean
</em>
</td>
<td>
<p>Enabled determines whether auditing configurations are applied to the nodes or not.<br />Will be defaulted to true, if AuditConfig is nil.</p>
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
<p>ConfigMapReferenceName is the name of the reference for the ConfigMap containing<br />auditing configuration to apply to shoot nodes.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="auditd">Auditd
</h3>


<p>
Auditd contains configuration for the audit daemon.
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
<code>auditRules</code></br>
<em>
string
</em>
</td>
<td>
<p>AuditRules contains the audit rules that will be placed under /etc/audit/rules.d.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="authmode">AuthMode
</h3>
<p><em>Underlying type: string</em></p>


<p>
(<em>Appears on:</em><a href="#tls">TLS</a>)
</p>

<p>
AuthMode is the type of authentication mode that can be used for the rsyslog relp connection to the target server.
</p>


<h3 id="loggingrule">LoggingRule
</h3>


<p>
(<em>Appears on:</em><a href="#rsyslogrelpconfig">RsyslogRelpConfig</a>)
</p>

<p>
LoggingRule contains options that determines which logs are sent to the target server.
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
string array
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
integer
</em>
</td>
<td>
<p>Severity determines which logs are sent to the target server based on their severity.</p>
</td>
</tr>
<tr>
<td>
<code>messageContent</code></br>
<em>
<a href="#messagecontent">MessageContent</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>MessageContent defines regular expressions for including and excluding logs based on their message content.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="messagecontent">MessageContent
</h3>


<p>
(<em>Appears on:</em><a href="#loggingrule">LoggingRule</a>)
</p>

<p>
MessageContent defines regular expressions for including and excluding logs based on their message content.
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
<code>regex</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Regex is a regular expression to match the message content of logs that should be sent to the target server.</p>
</td>
</tr>
<tr>
<td>
<code>exclude</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Exclude is a regular expression to match the message content of logs that should not be sent to the target server.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="rsyslogrelpconfig">RsyslogRelpConfig
</h3>


<p>
RsyslogRelpConfig configuration resource.
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
integer
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
<a href="#loggingrule">LoggingRule</a> array
</em>
</td>
<td>
<p>LoggingRules contain a list of LoggingRules that are used to determine which logs are<br />sent to the target server by the the rsyslog relp action.</p>
</td>
</tr>
<tr>
<td>
<code>tls</code></br>
<em>
<a href="#tls">TLS</a>
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
integer
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
integer
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
integer
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
boolean
</em>
</td>
<td>
<em>(Optional)</em>
<p>ReportSuspensionContinuation determines whether suspension continuation in the relp action<br />should be reported.</p>
</td>
</tr>
<tr>
<td>
<code>auditConfig</code></br>
<em>
<a href="#auditconfig">AuditConfig</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AuditConfig contains configuration that can be used to setup node level auditing so that audit logs<br />can be forwarded via rsyslog to the target RELP server.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="tls">TLS
</h3>


<p>
(<em>Appears on:</em><a href="#rsyslogrelpconfig">RsyslogRelpConfig</a>)
</p>

<p>
TLS contains options for the tls connection to the target server.
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
boolean
</em>
</td>
<td>
<p>Enabled determines whether TLS encryption should be used for the connection<br />to the target server.</p>
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
<p>SecretReferenceName is the name of the reference for the secret<br />containing the certificates for the TLS connection when encryption is enabled.</p>
</td>
</tr>
<tr>
<td>
<code>permittedPeer</code></br>
<em>
string array
</em>
</td>
<td>
<em>(Optional)</em>
<p>PermittedPeer is the name of the rsyslog relp permitted peer.<br />Only peers which have been listed in this parameter may be connected to.</p>
</td>
</tr>
<tr>
<td>
<code>authMode</code></br>
<em>
<a href="#authmode">AuthMode</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>AuthMode is the mode used for mutual authentication.<br />Possible values are "fingerprint" or "name".</p>
</td>
</tr>
<tr>
<td>
<code>tlsLib</code></br>
<em>
<a href="#tlslib">TLSLib</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TLSLib specifies the tls library that will be used by librelp on the shoot nodes.<br />If the field is omitted, the librelp default is used.<br />Possible values are "openssl" or "gnutls".</p>
</td>
</tr>

</tbody>
</table>


<h3 id="tlslib">TLSLib
</h3>
<p><em>Underlying type: string</em></p>


<p>
(<em>Appears on:</em><a href="#tls">TLS</a>)
</p>

<p>
TLSLib is the tls library that is used by the librelp library on the shoot's nodes.
</p>


