<?php GlobalHeader(); ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Define administrator"); ?></h1>

<?php if (GetBoolValue("NoUserEditActive")): ?>
<div class="flash mt-4">
  <p>
    <?php Translate("You have not defined an \"UserEditProviderType\", but the system also could not find any user from the configured user view provider."); ?><br>
    <?php Translate("You should now create a new user in your configured user authentication file."); ?>
  </p>
  <p class="mt-1"><?php Translate("If you think your configuration is incorrect, <a href=\"settings.php?firststart=1\">click here.</a>"); ?></p>
</div>
<?php endif; ?>

<?php if (GetBoolValue("DefaultUserCreated")): ?>
<div class="flash flash-success mt-4">
  <?php Translate("Could not find existing users with your configuration. Created default Administrator account."); ?>
  <p class="mt-1">
    <?php Translate("User"); ?>: <b>admin</b><br>
    <?php Translate("Password"); ?>: <b>admin</b>
  </p>
</div>
<p class="mt-4 text-sm">
  <?php Translate("You can <a href=\"login.php\"><b>login now.</b></a>"); ?>
</p>
<?php endif; ?>


<?php if (!GetBoolValue("DefaultUserCreated") && GetBoolValue("ShowUserSelection")): ?>
<div class="flash mt-4">
  <?php Translate("No administrator defined! Select the user for the administator privileges, please."); ?>
</div>

<form method="post" action="settings.php" class="mt-4 max-w-xl">
  <div class="mb-4">
    <label class="mb-1 block text-sm font-medium" for="selected_users"><?php Translate("User"); ?></label>
    <select name="selected_users[]" id="selected_users" class="input">
      <?php foreach(GetArrayValue("UserList") as $u): ?>
      <option value="<?php print($u->getEncodedName()); ?>"><?php print($u->getName()); ?></option>
      <?php endforeach; ?>
    </select>
  </div>
  <input type="hidden" name="setadmin" value="1">
  <button type="submit" name="saveadmin" value="<?php Translate("Save"); ?>" class="btn btn-primary"><?php Translate("Save"); ?></button>
</form>
<?php endif; ?>

<?php GlobalFooter(); ?>
