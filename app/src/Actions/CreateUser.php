<?php
namespace SvnAdmin\Actions;

class CreateUser extends Action
{
    public function execute(): void
    {
        //
        // Authentication
        //

        if (!$this->engine->isProviderActive(PROVIDER_USER_EDIT))
        {
        	$this->engine->forwardError(ERROR_INVALID_MODULE);
        }

        $this->engine->checkUserAuthentication(true, ACL_MOD_USER, ACL_ACTION_ADD);

        //
        // HTTP Request Vars
        //

        $username = get_request_var('username');
        $password = get_request_var('password');
        $password2 = get_request_var('password2');

        // Check required fields.
        if ($username == NULL || $password == NULL || $password2 == NULL)
        {
        	$this->engine->addException(new ValidationException(tr("You have to fill out all fields.")));
        }
        else if ($password != $password2)
        {
        	$this->engine->addException(new ValidationException(tr("The password's doesn't match each other.")));
        }
        else
        {
          // Create user object.
          $u = new \SvnAdmin\Entities\User;
          $u->id = $username;
          $u->name = $username;
          $u->password = $password;

          try {
        	  // Create the user now.
        	  $b = $this->engine->getUserEditProvider()->addUser($u);
        	  if($b)
        	  {
        	    $this->engine->getUserEditProvider()->save();
        	    $this->engine->addMessage(tr("The user %0 has been created successfully.", array($username)));
        	  }
        	  else
        	  {
        	  	$this->engine->addException(new Exception(tr("An unknown error occured. Check your configuration, please.")));
        	  }
          }
          catch (Exception $ex) {
          	$this->engine->addException($ex);
          }
        }
    }
}
