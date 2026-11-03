<?php
namespace SvnAdmin\I18n;

/**
 * Represents one available translation locale (entry of "translations/index.cfg").
 *
 * @author Manuel Freiholz
 */
class Locale
{
  public $locale;
  public $name;
  public $author;

  public function getLocale()
  {
    return $this->locale;
  }

  public function getName()
  {
    return $this->name;
  }

  public function getAuthor()
  {
    return $this->author;
  }
}
