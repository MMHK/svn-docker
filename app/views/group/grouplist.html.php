<?php GlobalHeader(); ?>

<?php /* 全選改用 assets/app.js：任何 id 以 selectall 開頭的核取方塊會選取同表格內的項目 */ ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Group management"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("Here you can see a list of all groups, which are defined in your subversion configuration."); ?></p>

<?php HtmlFilterBox("grouplist", 1); ?>

<form id="grouplist" action="grouplist.php" method="POST">
<div class="box mb-6 overflow-hidden">
	<div class="box-header"><?php Translate("Group"); ?></div>
	<div class="overflow-x-auto">
		<table class="w-full text-sm">
			<thead class="bg-canvas-subtle text-left text-fg-muted">
				<tr>
					<th scope="col" class="w-12 px-3 py-2 font-medium"><?php if (IsProviderActive(PROVIDER_GROUP_EDIT)  && HasAccess(ACL_MOD_GROUP, ACL_ACTION_DELETE)) { ?><input type="checkbox" id="selectall" aria-label="<?php Translate("Select all"); ?>"><?php } ?></th>
					<th scope="col" class="px-3 py-2 font-medium"><?php Translate("Group"); ?></th>
				</tr>
			</thead>

			<tbody>
				<?php foreach (GetArrayValue("GroupList") as $g) { ?>
				<tr class="box-row hover:bg-canvas-subtle">
					<td class="px-3 py-2"><?php if (IsProviderActive(PROVIDER_GROUP_EDIT) && HasAccess(ACL_MOD_GROUP, ACL_ACTION_DELETE)) { ?><input type="checkbox" name="selected_groups[]" value="<?php print($g->name); ?>" aria-label="<?php print($g->name); ?>"><?php } ?></td>
					<td class="px-3 py-2"><a class="text-accent-fg hover:underline" href="groupview.php?groupname=<?php print($g->getEncodedName()); ?>"><?php print($g->name); ?></a></td>
				</tr>
				<?php } ?>
			</tbody>

			<tfoot>
				<tr>
					<td colspan="2" class="px-3 py-3">
						<?php if (IsProviderActive(PROVIDER_GROUP_EDIT) && HasAccess(ACL_MOD_GROUP, ACL_ACTION_DELETE)) { ?>
						<button type="submit" name="delete" value="<?php Translate('Delete'); ?>" class="btn btn-danger" onclick="return deletionPrompt('<?php Translate("Are you sure?"); ?>');"><?php Translate('Delete'); ?></button>
						<?php } ?>
					</td>
				</tr>
			</tfoot>
		</table>
	</div>
</div>
</form>

<?php GlobalFooter(); ?>
