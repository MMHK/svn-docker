<?php GlobalHeader(); ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Change password of user"); ?> <?php PrintStringValue("Username"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("Change the password of your subversion account."); ?></p>
<div class="max-w-xl">
	<form method="POST" action="userchangepass.php">
		<input type="hidden" name="username" value="<?php PrintStringValue("Username"); ?>">

		<div class="mb-4">
		    <label class="mb-1 block text-sm font-medium" for="password"><?php Translate("New password"); ?>:</label>
		    <input type="password" name="password" id="password" class="input" autocomplete="new-password">
		</div>

		<div class="mb-4">
			<label class="mb-1 block text-sm font-medium" for="password2"><?php Translate("Re-type password"); ?>:</label>
			<input type="password" name="password2" id="password2" class="input" autocomplete="new-password">
		</div>

		<div class="mb-4">
			<button type="submit" name="changepass" value="<?php Translate("Change password"); ?>" class="btn btn-primary"><?php Translate("Change password"); ?></button>
		</div>

	</form>
</div>

<?php GlobalFooter(); ?>
