<?php GlobalHeader(); ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Settings"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("Read-only system information. The configuration is defined in data/config.ini."); ?></p>

<!-- System information -->
<div class="box mb-6 overflow-hidden" id="tbl_system">
  <div class="box-header"><?php Translate("System information"); ?></div>
  <div class="overflow-x-auto">
    <table class="w-full text-sm">
      <tbody>
        <tr class="box-row hover:bg-canvas-subtle">
          <th scope="row" class="w-1/2 px-4 py-2 text-left font-medium text-fg-muted"><?php Translate("Application version"); ?></th>
          <td class="px-4 py-2"><?php PrintStringValue("AppVersion"); ?></td>
        </tr>
        <tr class="box-row hover:bg-canvas-subtle">
          <th scope="row" class="w-1/2 px-4 py-2 text-left font-medium text-fg-muted"><?php Translate("PHP version"); ?></th>
          <td class="px-4 py-2"><?php PrintStringValue("PHPVersion"); ?></td>
        </tr>
        <tr class="box-row hover:bg-canvas-subtle">
          <th scope="row" class="w-1/2 px-4 py-2 text-left font-medium text-fg-muted"><?php Translate("SVN version"); ?></th>
          <td class="px-4 py-2"><?php PrintStringValue("SvnVersion"); ?></td>
        </tr>
      </tbody>
    </table>
  </div>
</div>

<!-- Subversion paths -->
<div class="box mb-6 overflow-hidden" id="tbl_subversion">
  <div class="box-header"><?php Translate("Subversion settings"); ?></div>
  <div class="overflow-x-auto">
    <table class="w-full text-sm">
      <tbody>
        <tr class="box-row hover:bg-canvas-subtle">
          <th scope="row" class="w-1/2 px-4 py-2 text-left font-medium text-fg-muted"><?php Translate("Subversion authorization file"); ?></th>
          <td class="px-4 py-2"><code class="rounded bg-canvas-subtle px-1 py-0.5 text-xs"><?php PrintStringValue("SVNAuthFile"); ?></code><br><small><b><?php PrintStringValue("SVNAuthFileState"); ?></b></small></td>
        </tr>
        <tr class="box-row hover:bg-canvas-subtle">
          <th scope="row" class="w-1/2 px-4 py-2 text-left font-medium text-fg-muted"><?php Translate("User authentication file (SVNUserFile)"); ?></th>
          <td class="px-4 py-2"><code class="rounded bg-canvas-subtle px-1 py-0.5 text-xs"><?php PrintStringValue("SVNUserFile"); ?></code><br><small><b><?php PrintStringValue("SVNUserFileState"); ?></b></small></td>
        </tr>
        <tr class="box-row hover:bg-canvas-subtle">
          <th scope="row" class="w-1/2 px-4 py-2 text-left font-medium text-fg-muted"><?php Translate("Parent directory of the repositories (SVNParentPath)"); ?></th>
          <td class="px-4 py-2"><code class="rounded bg-canvas-subtle px-1 py-0.5 text-xs"><?php PrintStringValue("SVNParentPath"); ?></code><br><small><b><?php PrintStringValue("SVNParentPathState"); ?></b></small></td>
        </tr>
      </tbody>
    </table>
  </div>
</div>

<!-- Provider settings -->
<div class="box mb-6 overflow-hidden" id="tbl_providers">
  <div class="box-header"><?php Translate("Data providers"); ?></div>
  <div class="overflow-x-auto">
    <table class="w-full text-sm">
      <tbody>
        <?php foreach (GetArrayValue("ProviderTypes") as $type => $value): ?>
        <tr class="box-row hover:bg-canvas-subtle">
          <th scope="row" class="w-1/2 px-4 py-2 text-left font-medium text-fg-muted"><?php print($type); ?></th>
          <td class="px-4 py-2"><span class="label"><?php print($value); ?></span></td>
        </tr>
        <?php endforeach; ?>
      </tbody>
    </table>
  </div>
</div>

<?php GlobalFooter(); ?>
