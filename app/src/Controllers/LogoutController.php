<?php
namespace SvnAdmin\Controllers;

class LogoutController extends Controller
{
    public function handle(): \SvnAdmin\Http\Response
    {
        global $appEngine, $appTR;
        
        $this->checkAuthentication();
        $appTR->loadModule("logged_out");
        
        // Log the user off.
        $appEngine->handleAction('logout');
        
        // Forward to login page.
        return $this->redirect('login.php?loggedout=1');
    }
}
