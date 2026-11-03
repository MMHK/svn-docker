<?php
namespace SvnAdmin\Actions;

class LoginAuthentication extends Action
{
    public function execute(): void
    {
        //
        // HTTP Request Vars
        //

        $loginname = get_request_var('loginname');
        $loginpass = get_request_var('loginpass');

        //
        // Validation
        //

        if ($loginname == NULL || $loginpass == NULL)
        {
        	$this->engine->addException(new ValidationException(tr("You have to fill out all fields.")));
        }
        else
        {
        	try {
        		if ($this->engine->getAuthenticator() != null)
        		{
        			$u = new \SvnAdmin\Entities\User($loginname, $loginname);
        			$authOK = $this->engine->getAuthenticator()->authenticate( $u, $loginpass );
        			if ($authOK)
        			{
        				// Set session variable which indicates that the user is logged in.
        				\SvnAdmin\Http\Session::instance()->setUsername($loginname);
        				$this->engine->forward(PAGE_HOME, null, true);
        			}
        			else
        			{
        				$this->engine->addException(new ValidationException(tr("Wrong user/password combination.")));
        			}
        		}
        		else
        		{
        			// Authentication is deactivated!
        			// ...
        		}
        	}
        	catch (Exception $ex) {
        		$this->engine->addException($ex);
        	}
        }
    }
}
