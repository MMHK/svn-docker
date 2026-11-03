<?php GlobalHeader(); ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Access-Path"); ?>: <?php PrintStringValue("AccessPath"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("Assigned users and groups to this access-path."); ?></p>

<h2 class="mb-2 mt-6 text-xl font-semibold text-fg-default"><?php Translate("Assigned users"); ?></h2>
<form action="accesspathview.php?accesspath=<?php PrintStringValue("AccessPathEncoded"); ?>" method="POST">
	<input type="hidden" name="selected_accesspaths[]" value="<?php PrintStringValue("AccessPath"); ?>">

	<?php HtmlFilterBox("accesspathviewlist", 1); ?>

	<div class="box mb-6 overflow-hidden">
		<div class="overflow-x-auto">
			<table id="accesspathviewlist" class="w-full text-sm">
			<thead class="bg-canvas-subtle text-left text-fg-muted">
				<tr>
					<th scope="col" class="w-12 px-3 py-2 font-medium">#</th>
					<th scope="col" class="w-64 px-3 py-2 font-medium"><?php Translate("User"); ?></th>
					<th scope="col" class="px-3 py-2 font-medium"><?php Translate("Permission"); ?></th>
				</tr>
			</thead>

			<tbody>
			<?php foreach(GetArrayValue("UserList") as $u) : ?>
			<tr class="box-row hover:bg-canvas-subtle">
				<td class="px-3 py-2"><?php if(IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_UNASSIGN)): ?><input type="checkbox" name="selected_users[]" value="<?php print($u->getName()); ?>" aria-label="<?php print($u->getName()); ?>"><?php endif; ?></td>
				<td class="px-3 py-2"><a class="text-accent-fg hover:underline" href="userview.php?username=<?php print($u->getEncodedName()); ?>"><?php print($u->getName()); ?></a></td>
				<td class="px-3 py-2"><span class="label"><?php print($u->getPermission()); ?></span></td>
			</tr>
			<?php endforeach; ?>
			</tbody>

			<?php if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT)) : ?>
			<tfoot>
				<tr>
					<td colspan="3" class="px-3 py-3">
						<div class="flex flex-wrap items-center justify-between gap-3">
							<div>
								<?php if (HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_UNASSIGN)) : ?>
								<button type="submit" name="unassign" value="<?php Translate("Unassign"); ?>" class="btn"><?php Translate("Unassign"); ?></button>
								<?php endif; ?>
							</div>
							<div class="flex flex-wrap items-center justify-end gap-2">
								<?php if (HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_ASSIGN)) : ?>
								<label class="sr-only" for="selected_users"><?php Translate("User"); ?></label>
								<select name="selected_users[]" id="selected_users" class="input w-auto">
									<option value="">--- <?php Translate("User"); ?> ---</option>
									<?php foreach (GetArrayValue("UserListAll") as $u) : ?>
									<option value="<?php print($u->getName()); ?>"><?php print($u->getName()); ?></option>
									<?php endforeach; ?>
								</select>

								<label class="sr-only" for="permission"><?php Translate("Permission"); ?></label>
								<select name="permission" id="permission" class="input w-auto">
									<option value="<?php PrintStringValue("PermNone"); ?>"><?php Translate("No permission"); ?></option>
									<option value="<?php PrintStringValue("PermRead"); ?>"><?php Translate("Read only"); ?></option>
									<option value="<?php PrintStringValue("PermReadWrite"); ?>"><?php Translate("Read &amp; Write"); ?></option>
								</select>
								<button type="submit" name="assign_permission" value="<?php Translate("Assign"); ?>" class="btn"><?php Translate("Assign"); ?></button>
								<?php endif; ?>
							</div>
						</div>
					</td>
				</tr>
			</tfoot>
			<?php endif; ?>
			</table>
		</div>
	</div>
</form>

<h2 class="mb-2 mt-6 text-xl font-semibold text-fg-default"><?php Translate("Assigned groups"); ?></h2>
<form action="accesspathview.php?accesspath=<?php PrintStringValue("AccessPathEncoded"); ?>" method="POST">
<input type="hidden" name="selected_accesspaths[]" value="<?php PrintStringValue("AccessPath"); ?>">

<?php HtmlFilterBox("assignedgrouplist", 1); ?>

<div class="box mb-6 overflow-hidden">
	<div class="overflow-x-auto">
		<table id="assignedgrouplist" class="w-full text-sm">
		<thead class="bg-canvas-subtle text-left text-fg-muted">
		<tr>
			<th scope="col" class="w-12 px-3 py-2 font-medium">#</th>
			<th scope="col" class="w-64 px-3 py-2 font-medium"><?php Translate("Group"); ?></th>
			<th scope="col" class="px-3 py-2 font-medium"><?php Translate("Permission"); ?></th>
		</tr>
		</thead>
		<tbody>
		<?php foreach(GetArrayValue("GroupList") as $g): ?>
		<tr class="box-row hover:bg-canvas-subtle">
			<td class="px-3 py-2"><?php if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_UNASSIGN)): ?><input type="checkbox" name="selected_groups[]" value="<?php print($g->getName()); ?>" aria-label="<?php print($g->getName()); ?>"><?php endif; ?></td>
			<td class="px-3 py-2"><a class="text-accent-fg hover:underline" href="groupview.php?groupname=<?php print($g->getEncodedName()); ?>"><?php print($g->getName()); ?></a></td>
			<td class="px-3 py-2"><span class="label"><?php print($g->getPermission()); ?></span></td>
		</tr>
		<?php endforeach; ?>
		</tbody>
		<?php if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT)): ?>
		<tfoot>
		<tr>
			<td colspan="3" class="px-3 py-3">
				<div class="flex flex-wrap items-center justify-between gap-3">
					<div>
						<?php if (HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_UNASSIGN)): ?>
						<button type="submit" name="unassign" value="<?php Translate("Unassign"); ?>" class="btn"><?php Translate("Unassign"); ?></button>
						<?php endif; ?>
					</div>
					<div class="flex flex-wrap items-center justify-end gap-2">
						<?php if (HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_ASSIGN)): ?>
						<label class="sr-only" for="selected_groups"><?php Translate("Group"); ?></label>
						<select name="selected_groups[]" id="selected_groups" class="input w-auto">
							<option value="">--- <?php Translate("Group"); ?> ---</option>
							<?php foreach(GetArrayValue("GroupListAll") as $g): ?>
							<option value="<?php print($g->getName()); ?>"><?php print($g->getName()); ?></option>
							<?php endforeach; ?>
						</select>
						<label class="sr-only" for="group_permission"><?php Translate("Permission"); ?></label>
						<select name="permission" id="group_permission" class="input w-auto">
							<option value="<?php PrintStringValue("PermNone"); ?>"><?php Translate("No permission"); ?></option>
							<option value="<?php PrintStringValue("PermRead"); ?>"><?php Translate("Read only"); ?></option>
							<option value="<?php PrintStringValue("PermReadWrite"); ?>"><?php Translate("Read &amp; Write"); ?></option>
						</select>
						<button type="submit" name="assign_permission" value="<?php Translate("Assign"); ?>" class="btn"><?php Translate("Assign"); ?></button>
						<?php endif; ?>
					</div>
				</div>
			</td>
		</tr>
		</tfoot>
		<?php endif; ?>
		</table>
	</div>
</div>
</form>

<?php GlobalFooter(); ?>
