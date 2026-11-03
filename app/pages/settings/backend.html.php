<?php GlobalHeader(); ?>

<h1><?php Translate("Settings"); ?></h1>
<p class="hdesc"><?php Translate("Read-only system information. The configuration is defined in data/config.ini."); ?></p>

<!-- System information -->
<table class="datatable settings" id="tbl_system">
  <colgroup>
    <col width="50%">
    <col width="50%">
  </colgroup>
  <thead>
    <tr>
      <th colspan="2"><?php Translate("System information"); ?></th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><?php Translate("Application version"); ?></td>
      <td><?php PrintStringValue("AppVersion"); ?></td>
    </tr>
    <tr>
      <td><?php Translate("PHP version"); ?></td>
      <td><?php PrintStringValue("PHPVersion"); ?></td>
    </tr>
    <tr>
      <td><?php Translate("SVN version"); ?></td>
      <td><?php PrintStringValue("SvnVersion"); ?></td>
    </tr>
  </tbody>
</table>
<br>

<!-- Subversion paths -->
<table class="datatable settings" id="tbl_subversion">
  <colgroup>
    <col width="50%">
    <col width="50%">
  </colgroup>
  <thead>
    <tr>
      <th colspan="2"><?php Translate("Subversion settings"); ?></th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><?php Translate("Subversion authorization file"); ?></td>
      <td><code><?php PrintStringValue("SVNAuthFile"); ?></code><br><small><b><?php PrintStringValue("SVNAuthFileState"); ?></b></small></td>
    </tr>
    <tr>
      <td><?php Translate("User authentication file (SVNUserFile)"); ?></td>
      <td><code><?php PrintStringValue("SVNUserFile"); ?></code><br><small><b><?php PrintStringValue("SVNUserFileState"); ?></b></small></td>
    </tr>
    <tr>
      <td><?php Translate("Parent directory of the repositories (SVNParentPath)"); ?></td>
      <td><code><?php PrintStringValue("SVNParentPath"); ?></code><br><small><b><?php PrintStringValue("SVNParentPathState"); ?></b></small></td>
    </tr>
  </tbody>
</table>
<br>

<!-- Provider settings -->
<table class="datatable settings" id="tbl_providers">
  <colgroup>
    <col width="50%">
    <col width="50%">
  </colgroup>
  <thead>
    <tr>
      <th colspan="2"><?php Translate("Data providers"); ?></th>
    </tr>
  </thead>
  <tbody>
    <?php foreach (GetArrayValue("ProviderTypes") as $type => $value): ?>
    <tr>
      <td><?php print($type); ?></td>
      <td><?php print($value); ?></td>
    </tr>
    <?php endforeach; ?>
  </tbody>
</table>
<br>

<?php GlobalFooter(); ?>
