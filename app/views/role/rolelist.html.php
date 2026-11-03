<?php GlobalHeader(); ?>

<?php if (/*IsAuthActive? && */ HasAccess(ACL_MOD_ROLE, ACL_ACTION_VIEW)) : ?>
<div id="rolelist">
  <h1 class="mb-4 text-2xl font-semibold text-fg-default"><?php Translate('Available roles'); ?></h1>
  <div class="box overflow-hidden">
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="bg-canvas-subtle text-left text-fg-muted">
          <tr>
            <th scope="col" class="px-3 py-2 font-medium"><?php Translate('Role'); ?></th>
            <th scope="col" class="px-3 py-2 font-medium"><?php Translate('Description'); ?></th>
          </tr>
        </thead>
        <tbody>
        <?php foreach(GetArrayValue('RoleList') as $r): ?>
        <tr class="box-row hover:bg-canvas-subtle">
          <td class="px-3 py-2"><span class="label"><?php print($r->getTranslatedName()); ?></span></td>
          <td class="px-3 py-2 text-fg-muted"><?php print($r->getTranslatedDescription()); ?></td>
        </tr>
        <?php endforeach; ?>
        </tbody>
      </table>
    </div>
  </div>
</div>
<?php endif; ?>

<?php GlobalFooter(); ?>
