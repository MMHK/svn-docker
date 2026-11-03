<?php
namespace SvnAdmin\Actions;

class Logout extends Action
{
    public function execute(): void
    {
        // Redirect, if the user authentication is not active.
        if( !$this->engine->isAuthenticationActive() )
        {
          header('Location: index.php');
        }

        // Destroy session now.
        \SvnAdmin\Http\Session::instance()->destroy();
    }
}
