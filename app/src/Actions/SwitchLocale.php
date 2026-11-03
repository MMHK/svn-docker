<?php
namespace SvnAdmin\Actions;

class SwitchLocale extends Action
{
    public function execute(): void
    {
        $requestedLocale = get_request_var("locale");

        // Set the new locale.
        if ($requestedLocale == NULL)
        {
          // No locale given...
        }
        else
        {
          $_COOKIE["locale"] = $requestedLocale;
          @setcookie("locale", $requestedLocale, time()+60*60*24*365); // 365 days
          header("Location: index.php");
        }
    }
}
