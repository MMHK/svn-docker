<?php GlobalHeader(); ?>

<?php /* 「顯示角色」改為頁內 <details>，不再用 AJAX + 對話框外掛 */ ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("User"); ?>: <?php PrintStringValue("Username"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("On this page you can see all assignments of the user."); ?></p>

<?php if (IsProviderActive(PROVIDER_USER_EDIT) && HasAccess(ACL_MOD_USER, ACL_ACTION_CHANGEPASS_OTHER)) { ?>
<!--
  Change password button
-->
<div class="mb-4 flex justify-end">
  <form method="get" action="userchangepass.php">
    <input type="hidden" name="username" value="<?php PrintStringValue("UsernameEncoded"); ?>">
    <button type="submit" class="btn"><?php Translate("Change password"); ?></button>
  </form>
</div>
<?php } ?>


<?php if (IsProviderActive(PROVIDER_AUTHENTICATION) && HasAccess(ACL_MOD_ROLE, ACL_ACTION_VIEW)) { ?>
<!--
  Roles of user
-->
<h2 class="mb-2 mt-6 text-xl font-semibold text-fg-default"><?php Translate("Roles of user"); ?></h2>
<form action="userview.php?username=<?php PrintStringValue("UsernameEncoded"); ?>" method="POST">
	<input type="hidden" name="selected_users[]" value="<?php PrintStringValue("Username"); ?>">
	<div class="box mb-6 overflow-hidden">
		<div class="overflow-x-auto">
			<table class="w-full text-sm">
				<thead class="bg-canvas-subtle text-left text-fg-muted">
					<tr>
						<th scope="col" class="w-12 px-3 py-2 font-medium">#</th>
						<th scope="col" class="px-3 py-2 font-medium"><?php Translate("Role"); ?></th>
						<th scope="col" class="px-3 py-2 font-medium"><?php Translate("Description"); ?></th>
					</tr>
				</thead>

				<tbody>
				<?php foreach (GetArrayValue("RoleList") as $r) { ?>
				<tr class="box-row hover:bg-canvas-subtle">
					<td class="px-3 py-2"><?php if (HasAccess(ACL_MOD_ROLE, ACL_ACTION_UNASSIGN)) { ?><input type="checkbox" name="selected_roles[]" value="<?php print($r->name); ?>" aria-label="<?php print($r->name); ?>"><?php } ?></td>
					<td class="px-3 py-2"><span class="label"><?php Translate($r->name); ?></span></td>
					<td class="px-3 py-2 text-fg-muted"><?php Translate($r->description); ?></td>
				</tr>
				<?php } ?>
				</tbody>

				<tfoot>
					<tr>
						<td colspan="3" class="px-3 py-3">
							<div class="flex flex-wrap items-start justify-between gap-3">
								<div>
									<?php if (HasAccess(ACL_MOD_ROLE, ACL_ACTION_UNASSIGN)) { ?>
									<button type="submit" name="unassign_role" value="<?php Translate("Unassign"); ?>" class="btn"><?php Translate("Unassign"); ?></button>
									<?php } ?>
								</div>
								<div class="flex flex-wrap items-center justify-end gap-2">
									<?php if (HasAccess(ACL_MOD_ROLE, ACL_ACTION_ASSIGN)) { ?>
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
													<?php foreach (GetArrayValue("RoleListAll") as $r) { ?>
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
										<?php foreach (GetArrayValue("RoleListAll") as $r) { ?>
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
<?php } ?>


<?php if (IsProviderActive(PROVIDER_GROUP_VIEW) && HasAccess(ACL_MOD_GROUP, ACL_ACTION_VIEW)) { ?>
<!--
  Groups of user
-->
<h2 class="mb-2 mt-6 text-xl font-semibold text-fg-default"><?php Translate("Groups of user"); ?></h2>
<form action="userview.php?username=<?php PrintStringValue("UsernameEncoded"); ?>" method="POST">
<input type="hidden" name="selected_users[]" value="<?php PrintStringValue("Username"); ?>">

<?php HtmlFilterBox("usergrouplist", 1); ?>

<div class="box mb-6 overflow-hidden">
	<div class="overflow-x-auto">
		<table id="usergrouplist" class="w-full text-sm">
			<thead class="bg-canvas-subtle text-left text-fg-muted">
				<tr>
					<th scope="col" class="w-12 px-3 py-2 font-medium">#</th>
					<th scope="col" class="px-3 py-2 font-medium"><?php Translate("Group"); ?></th>
				</tr>
			</thead>

			<tbody>
			<?php foreach (GetArrayValue("GroupList") as $g) { ?>
			<tr class="box-row hover:bg-canvas-subtle">
				<td class="px-3 py-2"><?php if (IsProviderActive(PROVIDER_GROUP_EDIT) && HasAccess(ACL_MOD_GROUP, ACL_ACTION_UNASSIGN)) { ?><input type="checkbox" name="selected_groups[]" value="<?php print($g->name); ?>" aria-label="<?php print($g->name); ?>"><?php } ?></td>
				<td class="px-3 py-2"><a class="text-accent-fg hover:underline" href="groupview.php?groupname=<?php print($g->getEncodedName()); ?>"><?php print($g->name); ?></a></td>
			</tr>
			<?php } ?>
			</tbody>

			<tfoot>
				<tr>
					<td colspan="2" class="px-3 py-3">
						<?php if (IsProviderActive(PROVIDER_GROUP_EDIT)) { ?>
						<div class="flex flex-wrap items-center justify-between gap-3">
							<div>
								<?php if (HasAccess(ACL_MOD_GROUP, ACL_ACTION_UNASSIGN)) {?>
								<button type="submit" name="unassign" value="<?php Translate("Unassign"); ?>" class="btn"><?php Translate("Unassign"); ?></button>
								<?php } ?>
							</div>
							<div class="flex flex-wrap items-center justify-end gap-2">
								<?php if (HasAccess(ACL_MOD_GROUP, ACL_ACTION_ASSIGN)) { ?>
								<label class="sr-only" for="selected_groups"><?php Translate("Group"); ?></label>
								<select name="selected_groups[]" id="selected_groups" class="input w-auto">
									<option value="">--- <?php Translate("Group"); ?> ---</option>
									<?php foreach (GetArrayValue("GroupListAll") as $g) { ?>
									<option value="<?php print($g->name); ?>"><?php print($g->name); ?></option>
									<?php } ?>
								</select>
								<button type="submit" name="assign_usergroup" value="<?php Translate("Assign"); ?>" class="btn"><?php Translate("Assign"); ?></button>
								<?php } ?>
							</div>
						</div>
						<?php } ?>
					</td>
				</tr>
			</tfoot>
		</table>
	</div>
</div>
</form>
<?php } ?>


<?php if (IsProviderActive(PROVIDER_ACCESSPATH_VIEW) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_VIEW)) { ?>
<!--
  Permissions of user
-->
<h2 class="mb-2 mt-6 text-xl font-semibold text-fg-default"><?php Translate("Permissions of users"); ?></h2>
<form action="userview.php?username=<?php PrintStringValue("UsernameEncoded"); ?>" method="POST">
<input type="hidden" name="selected_users[]" value="<?php PrintStringValue("Username"); ?>">

<?php HtmlFilterBox("userpermissionlist", 1); ?>

<div class="box mb-6 overflow-hidden">
	<div class="overflow-x-auto">
	<table id="userpermissionlist" class="w-full text-sm">
		<thead class="bg-canvas-subtle text-left text-fg-muted">
			<tr>
				<th scope="col" class="w-12 px-3 py-2 font-medium">#</th>
				<th scope="col" class="px-3 py-2 font-medium"><?php Translate("Access-Path"); ?></th>
				<th scope="col" class="px-3 py-2 font-medium"><?php Translate("Permission"); ?></th>
				<th scope="col" class="px-3 py-2 font-medium"><?php Translate("Inherit from Group"); ?></th>
			</tr>
		</thead>

		<tbody>
		<?php foreach (GetArrayValue("PathList") as $ap) { ?>
		<tr class="box-row hover:bg-canvas-subtle">
			<td class="px-3 py-2"><?php if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_UNASSIGN) && empty($ap->inherited)) { ?><input type="checkbox" name="selected_accesspaths[]" value="<?php print($ap->path); ?>" aria-label="<?php print($ap->path); ?>"><?php } ?></td>
			<td class="px-3 py-2"><a class="text-accent-fg hover:underline" href="accesspathview.php?accesspath=<?php print($ap->getEncodedPath()); ?>"><?php print($ap->path); ?></a></td>
			<td class="px-3 py-2"><span class="label"><?php print($ap->perm); ?></span></td>
			<td class="px-3 py-2 text-fg-muted"><?php print($ap->inherited); ?></td>
		</tr>
		<?php } ?>
		</tbody>

		<tfoot>
			<tr>
				<td colspan="4" class="px-3 py-3">
					<div class="flex flex-wrap items-center justify-between gap-3">
						<div>
							<?php if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_UNASSIGN)) { ?>
							<button type="submit" name="unassign_permission" value="<?php Translate("Unassign"); ?>" class="btn"><?php Translate("Unassign"); ?></button>
							<?php Translate("<b>Note:</b> You can not unassign permissions from user, which are inherited by a group."); ?>
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
<?php } ?>


<?php if (IsProviderActive(PROVIDER_AUTHENTICATION) && GetBoolValue("ProjectManager") && HasAccess(ACL_MOD_PROJECTMANAGER, ACL_ACTION_VIEW)) { ?>
<!--
  Project manager's access path list
-->
<h2 class="mb-2 mt-6 text-xl font-semibold text-fg-default"><?php Translate("User is project manager of:"); ?></h2>
<form action="userview.php?username=<?php PrintStringValue("UsernameEncoded"); ?>" method="POST">
<input type="hidden" name="selected_users[]" value="<?php PrintStringValue("Username"); ?>">
<div class="box mb-6 overflow-hidden">
	<div class="overflow-x-auto">
		<table class="w-full text-sm">
			<thead class="bg-canvas-subtle text-left text-fg-muted">
				<tr>
					<th scope="col" class="w-12 px-3 py-2 font-medium"></th>
					<th scope="col" class="px-3 py-2 font-medium"></th>
				</tr>
			</thead>

			<tbody>
				<?php foreach (GetArrayValue("RestrictedPathList") as $rp) { ?>
				<tr class="box-row hover:bg-canvas-subtle">
					<td class="px-3 py-2"><?php if (HasAccess(ACL_MOD_PROJECTMANAGER, ACL_ACTION_UNASSIGN)) { ?><input type="checkbox" name="selected_accesspaths[]" value="<?php print($rp->path); ?>" aria-label="<?php print($rp->path); ?>"><?php } ?></td>
					<td class="px-3 py-2"><a class="text-accent-fg hover:underline" href="accesspathview.php?accesspath=<?php print($rp->getEncodedPath()); ?>"><?php print($rp->path); ?></a> <small class="text-fg-muted">(<?php Translate("+All sub Access-Path's"); ?>)</small></td>
				</tr>
				<?php } ?>
			</tbody>

			<tfoot>
				<tr>
					<td colspan="2" class="px-3 py-3">
						<div class="flex flex-wrap items-center justify-between gap-3">
							<div>
								<?php if (HasAccess(ACL_MOD_PROJECTMANAGER, ACL_ACTION_UNASSIGN)) { ?>
								<button type="submit" name="unassign_projectmanager" value="<?php Translate("Unassign"); ?>" class="btn"><?php Translate("Unassign"); ?></button>
								<?php } ?>
							</div>
							<div></div>
						</div>
					</td>
				</tr>
			</tfoot>
		</table>
	</div>
</div>
</form>
<?php } ?>


<?php GlobalFooter(); ?>
