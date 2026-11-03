<?php GlobalHeader(); ?>

<?php /* 全選改用 assets/app.js：任何 id 以 selectall 開頭的核取方塊會選取同表格內的項目 */ ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Access-Path management"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("Here you can see a list of all access path, which are defined in your subversion configuration."); ?></p>

<?php HtmlFilterBox("accesspathlist", 1); ?>

<form action="accesspathslist.php" method="POST">
<div class="box mb-6 overflow-hidden">
	<div class="box-header"><?php Translate("Access-Path"); ?></div>
	<div class="overflow-x-auto">
		<table id="accesspathlist" class="w-full text-sm">
			<thead class="bg-canvas-subtle text-left text-fg-muted">
				<tr>
					<th scope="col" class="w-12 px-3 py-2 font-medium"><?php if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_DELETE)) : ?><input type="checkbox" id="selectall" aria-label="<?php Translate("Select all"); ?>"><?php endif; ?></th>
					<th scope="col" class="px-3 py-2 font-medium"><?php Translate("Access-Path"); ?></th>
				</tr>
			</thead>

			<tbody>
			<?php foreach (GetArrayValue("AccessPathList") as $ap) : ?>
			<tr class="box-row hover:bg-canvas-subtle">
				<td class="px-3 py-2"><?php if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_DELETE)) : ?><input type="checkbox" name="selected_accesspaths[]" value="<?php print($ap->getPath()); ?>" aria-label="<?php print($ap->getPath()); ?>"><?php endif; ?></td>
				<td class="px-3 py-2">
					<a class="text-accent-fg hover:underline" href="accesspathview.php?accesspath=<?php print($ap->getEncodedPath()); ?>"><?php print($ap->getPath()); ?></a><br>
					<small class="text-fg-muted"><?php Translate("Managers"); ?>: <?php print($ap->getManagersAsString()); ?></small>
				</td>
			</tr>
			<?php endforeach; ?>
			</tbody>

			<tfoot>
				<tr>
					<td colspan="2" class="px-3 py-3">
						<div class="flex flex-wrap items-center justify-between gap-3">
							<div>
								<?php if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_DELETE)) : ?>
								<button type="submit" name="delete" value="<?php Translate("Delete"); ?>" class="btn btn-danger" onclick="return deletionPrompt('<?php Translate("Are you sure?"); ?>');"><?php Translate("Delete"); ?></button>
								<?php endif; ?>
							</div>
							<div class="flex flex-wrap items-center justify-end gap-2">
								<?php if (IsProviderActive(PROVIDER_USER_VIEW) && HasAccess(ACL_MOD_PROJECTMANAGER, ACL_ACTION_ASSIGN)) : ?>
								<label class="sr-only" for="selected_users"><?php Translate("Set project manager"); ?></label>
								<select name="selected_users[]" id="selected_users" class="input w-auto">
									<option value="">--- <?php Translate("Set project manager"); ?> ---</option>
								<?php foreach (GetArrayValue("UserList") as $u) : ?>
									<option><?php print($u->name); ?></option>
								<?php endforeach; ?>
								</select>
								<button type="submit" name="assign_projectmanager" value="<?php Translate("Assign"); ?>" class="btn"><?php Translate("Assign"); ?></button>
								<?php endif; ?>
							</div>
						</div>
					</td>
				</tr>
			</tfoot>
		</table>
	</div>
</div>
</form>

<?php GlobalFooter(); ?>
