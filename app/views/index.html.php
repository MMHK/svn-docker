<?php GlobalHeader(); ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default">
	<?php Translate("Welcome"); ?>
	<?php if (IsUserLoggedIn()) { ?><span class="text-fg-muted"><?php SessionUsername(); ?></span><?php } ?>
</h1>

<div class="box mt-4">
  <div class="box-header"><?php Translate("General information"); ?></div>
  <div class="overflow-x-auto">
    <table class="w-full text-sm">
      <tbody>
        <tr class="box-row hover:bg-canvas-subtle">
          <th scope="row" class="w-56 px-4 py-2 text-left font-medium text-fg-muted"><?php Translate("Application version"); ?></th>
          <td class="px-4 py-2"><?php AppVersion(); ?> <small class="text-fg-muted">(<a class="text-accent-fg hover:underline" href="http://svnadmin.insanefactory.com/"><?php Translate("Check for updates"); ?></a>)</small></td>
        </tr>
        <tr class="box-row hover:bg-canvas-subtle">
          <th scope="row" class="w-56 px-4 py-2 text-left font-medium text-fg-muted"><?php Translate("PHP version"); ?></th>
          <td class="px-4 py-2"><?php PrintStringValue("PHPVersion"); ?></td>
        </tr>
        <?php if (IsUserLoggedIn()) { ?>
        <tr class="box-row hover:bg-canvas-subtle">
          <th scope="row" class="w-56 px-4 py-2 text-left font-medium text-fg-muted"><?php Translate("Logged in as"); ?></th>
          <td class="px-4 py-2"><?php SessionUsername(); ?></td>
        </tr>
        <?php } else { ?>
        <tr class="box-row hover:bg-canvas-subtle">
          <th scope="row" class="w-56 px-4 py-2 text-left font-medium text-fg-muted"><?php Translate("Logged in as"); ?></th>
          <td class="px-4 py-2"><?php Translate("Guest"); ?></td>
        </tr>
        <?php } ?>
        <?php if (IsProviderActive(PROVIDER_AUTHENTICATION)) { ?>
        <tr class="box-row hover:bg-canvas-subtle">
          <th scope="row" class="w-56 px-4 py-2 text-left font-medium text-fg-muted"><?php Translate("Roles of user"); ?></th>
          <td class="px-4 py-2">
            <ul class="space-y-1">
              <?php foreach (GetArrayValue("Roles") as $r) { ?>
              <li class="flex flex-wrap items-baseline gap-2">
                <span class="label"><?php Translate($r->name); ?></span>
                <i class="text-xs text-fg-muted"><?php Translate($r->description); ?></i>
              </li>
              <?php } ?>
            </ul>
          </td>
        </tr>
        <?php } ?>
      </tbody>
    </table>
  </div>
</div>

<?php GlobalFooter(); ?>
