<?php
namespace SvnAdmin\Actions;

class ChangePassword extends Action
{
    public function execute(): void
    {
        $this->engine->forwardInvalidModule(!$this->engine->isUserEditActive());
        $this->engine->checkUserAuthentication(true, ACL_MOD_USER, ACL_ACTION_CHANGEPASS);

        // Parameters.
        $password = get_request_var("password");
        $password2 = get_request_var("password2");
        $username = get_request_var("username");

        // Validation.
        $sessionUser = $this->engine->getSessionUsername();
        if ($sessionUser != NULL) {
          if ($sessionUser != $username) {
            $this->engine->checkUserAuthentication(true, ACL_MOD_USER, ACL_ACTION_CHANGEPASS_OTHER);
          }
        }

        if ($username == NULL) {
          $this->engine->addException(new ValidationException(tr("No username given.")));
        }
        else if ($password == NULL || $password2 == NULL) {
          $this->engine->addException(new ValidationException(tr("You have to fill out all fields.")));
        }
        else if($password != $password2) {
          $this->engine->addException(new ValidationException(tr("The password's doesn't match each other.")));
        }
        else {
          try {
            // Ok, change password now.
            $b = $this->engine->getUserEditProvider()->changePassword($username, $password);
            if ($b) {
              $this->engine->getUserEditProvider()->save();
              $this->engine->addMessage(tr("The password has been changed."));
            }
            else {
              throw new Exception(tr("An unknown error occured. Check your configuration, please."));
            }
          }
          catch (Exception $ex) {
            $this->engine->addException($ex);
          }
        }
    }
}
