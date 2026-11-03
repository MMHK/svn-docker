<?php GlobalHeader(); ?>

<?php /* 全選改用 assets/app.js；「顯示角色」改為頁內 <details>，不再用 AJAX + 對話框外掛 */ ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("User management"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("Here you can see a list of all users which can be authenticated by your subversion server."); ?></p>

<?php HtmlFilterBox("userlist", 1); ?>

<form action="userlist.php" method="POST">
	<div class="box mb-6 overflow-hidden">
		<div class="box-header"><?php Translate("User"); ?></div>
		<div class="overflow-x-auto">
			<table id="userlist" class="w-full text-sm">
				<thead class="bg-canvas-subtle text-left text-fg-muted">
					<tr>
						<th scope="col" class="w-12 px-3 py-2 font-medium">
							<?php if (HasAccess(ACL_MOD_USER, ACL_ACTION_DELETE) || HasAccess(ACL_MOD_ROLE,	ACL_ACTION_ASSIGN)) { ?>
							<input type="checkbox" id="selectall" aria-label="<?php Translate("Select all"); ?>">
							<?php } ?>
						</th>
						<th scope="col" class="px-3 py-2 font-medium">
							<?php Translate("User"); ?>
						</th>
					</tr>
				</thead>

				<tbody>
					<?php foreach (GetArrayValue("UserList") as $u) { ?>
					<tr class="box-row hover:bg-canvas-subtle">
						<td class="px-3 py-2">
							<?php if (HasAccess(ACL_MOD_USER, ACL_ACTION_DELETE) || HasAccess(ACL_MOD_ROLE,	ACL_ACTION_ASSIGN)) { ?>
							<input type="checkbox" name="selected_users[]" value="<?php print($u->name); ?>" aria-label="<?php print($u->name); ?>">
							<?php } ?>
						</td>
						<td class="px-3 py-2"><a class="text-accent-fg hover:underline" href="userview.php?username=<?php print($u->getEncodedName()); ?>"><?php print($u->name); ?></a></td>
					</tr>
					<?php } ?>
				</tbody>

				<tfoot>
					<tr>
						<td colspan="2" class="px-3 py-3">
							<div class="flex flex-wrap items-start justify-between gap-3">
								<div>
									<?php if (IsProviderActive(PROVIDER_USER_EDIT) && HasAccess(ACL_MOD_USER, ACL_ACTION_DELETE)) { ?>
									<button type="submit" name="delete" value="<?php Translate("Delete"); ?>" class="btn btn-danger" onclick="return deletionPrompt('<?php Translate("Are you sure?"); ?>');"><?php Translate("Delete"); ?></button>
									<?php } ?>
								</div>
								<div class="flex flex-wrap items-center justify-end gap-2">
									<?php if (IsProviderActive(PROVIDER_AUTHENTICATION) && HasAccess(ACL_MOD_ROLE, ACL_ACTION_ASSIGN)) { ?>
									<details class="text-left">
										<summary class="cursor-pointer text-xs text-accent-fg"><?php Translate("Show roles"); ?></summary>
										<div class="mt-2 overflow-x-auto rounded-md border border-border-default">
											<table class="w-full text-xs">
												<thead class="bg-canvas-subtle text-left text-fg-muted">
													<tr>
														<th scope="col" class="px-2 py-1 font-medium"><?php Translate("Role"); ?></th>
														<th scope="col" class="px-2 py-1 font-medium"><?php Translate("Description"); ?></th>
													</tr>
												</thead>
												<tbody>
													<?php foreach (GetArrayValue("RoleList") as $r) { ?>
													<tr class="box-row">
														<td class="px-2 py-1"><?php print($r->getTranslatedName()); ?></td>
														<td class="px-2 py-1 text-fg-muted"><?php print($r->getTranslatedDescription()); ?></td>
													</tr>
													<?php } ?>
												</tbody>
											</table>
										</div>
									</details>
									<label class="sr-only" for="selected_assign_role_name"><?php Translate("Role"); ?></label>
									<select name="selected_assign_role_name" id="selected_assign_role_name" class="input w-auto">
										<option value="">--- <?php Translate("Role"); ?> ---</option>
										<?php foreach (GetArrayValue("RoleList") as $r) { ?>
										<option value="<?php print($r->name); ?>"><?php Translate($r->name); ?></option>
										<?php } ?>
									</select>
									<button type="submit" name="assign_role" value="<?php Translate("Assign"); ?>" class="btn"><?php Translate("Assign"); ?></button>
									<?php } ?>
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
