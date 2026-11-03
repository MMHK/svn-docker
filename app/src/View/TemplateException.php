<?php
namespace SvnAdmin\View;

/**
 * Raised if the template engine finds an invalid template.
 *
 * @author Manuel Freiholz / www.gainwar.de
 */
class TemplateException extends \Exception
{
  public function __construct($message="", $code=0, ?\Exception $previous=null)
  {
    parent::__construct($message, $code, $previous);
  }
}
